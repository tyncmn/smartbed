// Package worker – Daily sleep summary worker.
// Runs a scheduled job to compute and persist nightly sleep summaries.
package worker

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

// DailySummaryWorker runs periodic sleep summary computation.
type DailySummaryWorker struct {
	interval time.Duration
	// sleepSvc could be injected here to compute summaries per user
	// For now it logs and can be extended with full user enumeration
}

// NewDailySummaryWorker creates a new DailySummaryWorker.
func NewDailySummaryWorker(interval time.Duration) *DailySummaryWorker {
	return &DailySummaryWorker{interval: interval}
}

// Run starts the daily summary loop until ctx is cancelled.
func (w *DailySummaryWorker) Run(ctx context.Context) {
	log.Info().Dur("interval", w.interval).Msg("Daily summary worker started")
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Daily summary worker stopped")
			return
		case t := <-ticker.C:
			log.Info().Time("tick", t).Msg("Daily summary job triggered")
			// TODO: enumerate all active users and call sleepSvc.GetSleepSummary
			// then persist to a daily_summaries table or cache in Redis
		}
	}
}
