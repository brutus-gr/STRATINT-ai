package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/STRATINT/stratint/internal/database"
	"github.com/STRATINT/stratint/internal/strategist"
)

// StrategyScheduler manages automatic execution of scheduled strategies
type StrategyScheduler struct {
	strategyRepo  *database.StrategyRepository
	strategist    *strategist.Strategist
	logger        *slog.Logger
	stopChan      chan struct{}
	checkInterval time.Duration
}

// NewStrategyScheduler creates a new strategy scheduler
func NewStrategyScheduler(
	strategyRepo *database.StrategyRepository,
	strategist *strategist.Strategist,
	logger *slog.Logger,
) *StrategyScheduler {
	return &StrategyScheduler{
		strategyRepo:  strategyRepo,
		strategist:    strategist,
		logger:        logger,
		stopChan:      make(chan struct{}),
		checkInterval: 1 * time.Minute, // Check every minute
	}
}

// Start begins the scheduler loop
func (s *StrategyScheduler) Start(ctx context.Context) {
	s.logger.Info("[STRATEGY SCHEDULER] Starting", "check_interval", s.checkInterval)
	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	// Run once immediately on start
	s.logger.Info("[STRATEGY SCHEDULER] Running initial check")
	s.checkAndRunStrategies(ctx)
	s.logger.Info("[STRATEGY SCHEDULER] Initial check complete")

	for {
		select {
		case <-ticker.C:
			s.logger.Info("[STRATEGY SCHEDULER] Ticker fired, checking for strategies")
			s.checkAndRunStrategies(ctx)
		case <-s.stopChan:
			s.logger.Info("[STRATEGY SCHEDULER] Stopped")
			return
		case <-ctx.Done():
			s.logger.Info("[STRATEGY SCHEDULER] Stopping due to context cancellation")
			return
		}
	}
}

// Stop stops the scheduler
func (s *StrategyScheduler) Stop() {
	close(s.stopChan)
}

// checkAndRunStrategies checks for strategies that need to run and executes them
func (s *StrategyScheduler) checkAndRunStrategies(ctx context.Context) {
	s.logger.Info("[STRATEGY SCHEDULER] Checking for scheduled strategies")
	strategies, err := s.strategyRepo.GetScheduledStrategies(ctx)
	if err != nil {
		s.logger.Error("[STRATEGY SCHEDULER] Failed to get scheduled strategies", "error", err)
		return
	}

	if len(strategies) == 0 {
		s.logger.Info("[STRATEGY SCHEDULER] No scheduled strategies due to run")
		return
	}

	s.logger.Info("[STRATEGY SCHEDULER] Found scheduled strategies to run", "count", len(strategies))

	for _, strategy := range strategies {
		s.logger.Info("Executing scheduled strategy",
			"strategy_id", strategy.ID,
			"name", strategy.Name,
			"interval", strategy.ScheduleInterval,
			"last_run_at", strategy.LastRunAt,
			"next_run_at", strategy.NextRunAt,
		)

		// Execute the strategy
		runID, err := s.strategist.ExecuteStrategy(ctx, strategy.ID)
		if err != nil {
			s.logger.Error("Failed to execute scheduled strategy",
				"strategy_id", strategy.ID,
				"name", strategy.Name,
				"error", err,
			)
			continue
		}

		s.logger.Info("Successfully started scheduled strategy run",
			"strategy_id", strategy.ID,
			"name", strategy.Name,
			"run_id", runID,
		)

		// Note: last_run_at and next_run_at are already updated atomically
		// by GetScheduledStrategies using UPDATE...RETURNING, so no need to update again
	}
}
