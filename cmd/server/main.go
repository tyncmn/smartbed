// Package main – SmartBed server entrypoint.
// Wires all components together: config, DB, Redis, MQTT, services, handlers, router, and workers.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"smartbed/internal/config"
	"smartbed/internal/db"
	"smartbed/internal/handler"
	"smartbed/internal/logger"
	"smartbed/internal/middleware"
	mqttclient "smartbed/internal/mqtt"
	redisclient "smartbed/internal/redis"
	"smartbed/internal/service"
	"smartbed/internal/worker"
)

func main() {
	// ── 1. Load configuration ────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	// ── 2. Init logger ────────────────────────────────────────────────────────
	logger.Init(cfg.Log.Level, cfg.Log.Pretty)
	log.Info().Str("env", cfg.App.Env).Msg("SmartBed starting")

	// ── 3. Connect to PostgreSQL ───────────────────────────────────────────────
	sqlDB, err := db.Connect(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("database connect failed")
	}
	defer sqlDB.Close()

	// ── 4. Run migrations ──────────────────────────────────────────────────────
	dsn := fmt.Sprintf("%s:%s@%s:%s/%s?sslmode=%s",
		cfg.Database.User, cfg.Database.Password,
		cfg.Database.Host, cfg.Database.Port,
		cfg.Database.Name, cfg.Database.SSLMode,
	)
	if err := db.RunMigrations(dsn, cfg.App.Migrations); err != nil {
		log.Fatal().Err(err).Msg("migrations failed")
	}

	// ── 5. Connect to Redis ────────────────────────────────────────────────────
	rdb, err := redisclient.New(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("redis connect failed")
	}
	defer rdb.Close()

	// ── 6. Connect to MQTT ────────────────────────────────────────────────────
	mqttClient, err := mqttclient.New(cfg)
	if err != nil {
		log.Warn().Err(err).Msg("MQTT connect failed; IoT features disabled")
		// Proceed without MQTT in development
		mqttClient = nil
	}
	if mqttClient != nil {
		defer mqttClient.Disconnect()
	}

	// ── 7. Load JWT keys ──────────────────────────────────────────────────────
	privateKeyPEM, err := os.ReadFile(cfg.JWT.PrivateKeyPath)
	if err != nil {
		log.Fatal().Err(err).Str("path", cfg.JWT.PrivateKeyPath).Msg("read private key failed")
	}
	publicKeyPEM, err := os.ReadFile(cfg.JWT.PublicKeyPath)
	if err != nil {
		log.Fatal().Err(err).Str("path", cfg.JWT.PublicKeyPath).Msg("read public key failed")
	}
	jwtSvc, err := middleware.NewJWTService(
		privateKeyPEM, publicKeyPEM,
		cfg.JWT.AccessTokenExpiryMins,
		cfg.JWT.RefreshTokenExpiryDays,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("JWT service init failed")
	}

	// ── 8. Wire services (dependency order) ───────────────────────────────────
	auditLogger := middleware.NewAuditLogger(sqlDB)
	notificationSvc := service.NewNotificationService(sqlDB)
	alertSvc := service.NewAlertService(sqlDB, notificationSvc)
	baselineSvc := service.NewBaselineService(sqlDB)
	riskEngine := service.NewRiskEngine(sqlDB, baselineSvc, alertSvc)
	ingestionSvc := service.NewIngestionService(sqlDB, riskEngine)
	sleepSvc := service.NewSleepAnalyticsService(sqlDB)
	authSvc := service.NewAuthService(sqlDB, jwtSvc)

	var deviceSvc *service.DeviceCommandService
	if mqttClient != nil {
		deviceSvc = service.NewDeviceCommandService(sqlDB, mqttClient, cfg.MQTT.ACKTimeoutSec)
	}

	protocolSvc := service.NewProtocolService(sqlDB, deviceSvc)
	protocolSvc.SetAuditFunc(func(ctx context.Context,
		actorID uuid.UUID,
		action string,
		entityID uuid.UUID,
		detail interface{}) {
	})

	// ── 9. Wire handlers ──────────────────────────────────────────────────────
	authHandler := handler.NewAuthHandler(authSvc)
	vitalsHandler := handler.NewVitalsHandler(ingestionSvc, sleepSvc)
	alertHandler := handler.NewAlertHandler(alertSvc)
	dashboardHandler := handler.DashboardHandlerWithDeps(sleepSvc, alertSvc)
	protocolHandler := handler.NewProtocolHandler(protocolSvc, auditLogger)

	var deviceHandler *handler.DeviceHandler
	if deviceSvc != nil {
		deviceHandler = handler.NewDeviceHandler(deviceSvc, auditLogger)
	}

	// ── 10. Build Gin router ──────────────────────────────────────────────────
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.ErrorHandler())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	// Health check (unauthenticated)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "smartbed"})
	})

	v1 := r.Group("/api/v1")
	{
		// ── Auth (public) ──────────────────────────────────────────────────
		v1.POST("/auth/login", authHandler.Login)
		v1.POST("/auth/refresh", authHandler.Refresh)

		// ── Authenticated routes ───────────────────────────────────────────
		auth := v1.Group("", jwtSvc.Authenticate())
		{
			// Vitals ingestion (operators and devices)
			auth.POST("/vitals/ingest",
				middleware.RequireRoles(middleware.RoleOperatorAll()...),
				vitalsHandler.Ingest,
			)
			// Vitals read (live polling + sleep summary)
			auth.GET("/vitals/latest",
				middleware.RequireRoles(middleware.RoleClinicalAll()...),
				vitalsHandler.GetLatestVitals,
			)
			auth.GET("/vitals/sleep-summary",
				middleware.RequireRoles(middleware.RoleClinicalAll()...),
				vitalsHandler.GetVitalsSleepSummary,
			)

			// Alerts
			auth.GET("/alerts",
				middleware.RequireRoles(middleware.RoleClinicalAll()...),
				alertHandler.GetAlerts,
			)
			auth.PUT("/alerts/:id/acknowledge",
				middleware.RequireRoles(middleware.RoleClinicalAll()...),
				alertHandler.AcknowledgeAlert,
			)

			// Dashboard
			auth.GET("/users/:id/current-status",
				middleware.RequireRoles(middleware.RoleClinicalAll()...),
				dashboardHandler.GetUserCurrentStatus,
			)
			auth.GET("/users/:id/sleep-summary",
				middleware.RequireRoles(middleware.RoleClinicalAll()...),
				dashboardHandler.GetSleepSummary,
			)

			// Protocols (doctors only)
			auth.POST("/protocols",
				middleware.RequireRoles(middleware.RoleOnlyDoctor()...),
				protocolHandler.CreateProtocol,
			)
			auth.PUT("/protocols/:id/state",
				middleware.RequireRoles(middleware.RoleOnlyDoctor()...),
				protocolHandler.UpdateProtocolState,
			)

			// Device commands (doctors + admins)
			if deviceHandler != nil {
				auth.POST("/device-commands/:deviceId/execute",
					middleware.RequireRoles(middleware.RoleAdminDoctor()...),
					deviceHandler.ExecuteCommand,
				)
				auth.GET("/devices/:deviceId/status",
					middleware.RequireRoles(middleware.RoleAdminDoctor()...),
					deviceHandler.GetDeviceStatus,
				)
			}
		}
	}

	// ── 11. Start background workers ──────────────────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if mqttClient != nil && deviceSvc != nil {
		mqttWorker := worker.NewMQTTACKWorker(mqttClient, deviceSvc)
		go mqttWorker.Run(ctx)
	}

	dailyWorker := worker.NewDailySummaryWorker(24 * time.Hour)
	go dailyWorker.Run(ctx)

	// ── 12. Start HTTP server ─────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info().Str("addr", srv.Addr).Msg("HTTP server started")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	// ── 13. Graceful shutdown ─────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down...")

	cancel() // Stop background workers

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Server forced shutdown")
	}
	log.Info().Msg("SmartBed stopped")
}
