// Package logger configures a zerolog-based structured logger.
package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Init sets up the global zerolog logger based on config values.
func Init(levelStr string, pretty bool) {
	level, err := zerolog.ParseLevel(levelStr)
	if err != nil {
		level = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(level)
	zerolog.TimeFieldFormat = time.RFC3339

	var writer io.Writer = os.Stdout
	if pretty {
		writer = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05",
		}
	}
	log.Logger = zerolog.New(writer).With().Timestamp().Caller().Logger()
}

// Get returns the global zerolog logger.
func Get() zerolog.Logger {
	return log.Logger
}
