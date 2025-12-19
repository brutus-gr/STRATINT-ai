package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/STRATINT/stratint/internal/database"
	"github.com/STRATINT/stratint/internal/forecaster"
)

// ForecastScheduler manages automatic execution of scheduled forecasts
type ForecastScheduler struct {
	forecastRepo  *database.ForecastRepository
	forecaster    *forecaster.Forecaster
	logger        *slog.Logger
	stopChan      chan struct{}
	checkInterval time.Duration
}

// NewForecastScheduler creates a new forecast scheduler
func NewForecastScheduler(
	forecastRepo *database.ForecastRepository,
	forecaster *forecaster.Forecaster,
	logger *slog.Logger,
) *ForecastScheduler {
	return &ForecastScheduler{
		forecastRepo:  forecastRepo,
		forecaster:    forecaster,
		logger:        logger,
		stopChan:      make(chan struct{}),
		checkInterval: 1 * time.Minute, // Check every minute
	}
}

// Start begins the scheduler loop
func (s *ForecastScheduler) Start(ctx context.Context) {
	s.logger.Info("Starting forecast scheduler", "check_interval", s.checkInterval)
	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	// Run once immediately on start
	s.checkAndRunForecasts(ctx)

	for {
		select {
		case <-ticker.C:
			s.checkAndRunForecasts(ctx)
		case <-s.stopChan:
			s.logger.Info("Forecast scheduler stopped")
			return
		case <-ctx.Done():
			s.logger.Info("Forecast scheduler stopping due to context cancellation")
			return
		}
	}
}

// Stop stops the scheduler
func (s *ForecastScheduler) Stop() {
	close(s.stopChan)
}

// checkAndRunForecasts checks for forecasts that need to run and executes them
func (s *ForecastScheduler) checkAndRunForecasts(ctx context.Context) {
	forecasts, err := s.forecastRepo.GetScheduledForecasts(ctx)
	if err != nil {
		s.logger.Error("Failed to get scheduled forecasts", "error", err)
		return
	}

	if len(forecasts) == 0 {
		s.logger.Debug("No scheduled forecasts due to run")
		return
	}

	s.logger.Info("Found scheduled forecasts to run", "count", len(forecasts))

	for _, forecast := range forecasts {
		s.logger.Info("Executing scheduled forecast",
			"forecast_id", forecast.ID,
			"name", forecast.Name,
			"interval", forecast.ScheduleInterval,
			"last_run_at", forecast.LastRunAt,
			"next_run_at", forecast.NextRunAt,
		)

		// Execute the forecast
		runID, err := s.forecaster.ExecuteForecast(ctx, forecast.ID)
		if err != nil {
			s.logger.Error("Failed to execute scheduled forecast",
				"forecast_id", forecast.ID,
				"name", forecast.Name,
				"error", err,
			)
			continue
		}

		s.logger.Info("Successfully started scheduled forecast run",
			"forecast_id", forecast.ID,
			"name", forecast.Name,
			"run_id", runID,
		)

		// Note: last_run_at and next_run_at are already updated atomically
		// by GetScheduledForecasts using UPDATE...RETURNING, so no need to update again
	}
}
