package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/STRATINT/stratint/internal/api"
	"github.com/STRATINT/stratint/internal/database"
)

// SummaryScheduler manages automatic execution of scheduled summaries
type SummaryScheduler struct {
	summaryRepo     *database.SummaryRepository
	summaryExecutor *api.SummaryExecutor
	logger          *slog.Logger
	stopChan        chan struct{}
	checkInterval   time.Duration
}

// NewSummaryScheduler creates a new summary scheduler
func NewSummaryScheduler(
	summaryRepo *database.SummaryRepository,
	summaryExecutor *api.SummaryExecutor,
	logger *slog.Logger,
) *SummaryScheduler {
	return &SummaryScheduler{
		summaryRepo:     summaryRepo,
		summaryExecutor: summaryExecutor,
		logger:          logger,
		stopChan:        make(chan struct{}),
		checkInterval:   1 * time.Minute, // Check every minute
	}
}

// Start begins the scheduler loop
func (s *SummaryScheduler) Start(ctx context.Context) {
	s.logger.Info("Starting summary scheduler", "check_interval", s.checkInterval)
	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	// Run once immediately on start
	s.checkAndRunSummaries(ctx)

	for {
		select {
		case <-ticker.C:
			s.checkAndRunSummaries(ctx)
		case <-s.stopChan:
			s.logger.Info("Summary scheduler stopped")
			return
		case <-ctx.Done():
			s.logger.Info("Summary scheduler stopping due to context cancellation")
			return
		}
	}
}

// Stop stops the scheduler
func (s *SummaryScheduler) Stop() {
	close(s.stopChan)
}

// checkAndRunSummaries checks for summaries that need to run and executes them
func (s *SummaryScheduler) checkAndRunSummaries(ctx context.Context) {
	summaries, err := s.summaryRepo.List(ctx)
	if err != nil {
		s.logger.Error("Failed to get summaries", "error", err)
		return
	}

	now := time.Now()
	currentTime := now.Format("15:04")

	for _, summary := range summaries {
		// Skip if not active or no time_of_day set
		if !summary.Active || summary.TimeOfDay == nil || *summary.TimeOfDay == "" {
			continue
		}

		// Check if this summary should run now
		if *summary.TimeOfDay != currentTime {
			continue
		}

		// Check if we already ran today
		if summary.LastRunAt != nil {
			lastRun := *summary.LastRunAt
			if lastRun.Year() == now.Year() && lastRun.YearDay() == now.YearDay() {
				s.logger.Debug("Summary already ran today, skipping",
					"summary_id", summary.ID,
					"name", summary.Name,
					"last_run_at", lastRun.Format(time.RFC3339))
				continue
			}
		}

		s.logger.Info("Executing scheduled summary",
			"summary_id", summary.ID,
			"name", summary.Name,
			"time_of_day", *summary.TimeOfDay,
			"last_run_at", summary.LastRunAt,
		)

		// Execute the summary
		runID, err := s.summaryExecutor.Execute(ctx, summary.ID)
		if err != nil {
			s.logger.Error("Failed to execute scheduled summary",
				"summary_id", summary.ID,
				"name", summary.Name,
				"error", err,
			)
			continue
		}

		s.logger.Info("Successfully started scheduled summary run",
			"summary_id", summary.ID,
			"name", summary.Name,
			"run_id", runID,
		)
	}
}
