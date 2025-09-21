package main

import (
	"time"

	"github.com/ericogr/chimera-cards/internal/game"
	"github.com/ericogr/chimera-cards/internal/logging"
	"github.com/ericogr/chimera-cards/internal/service"
)

// startTimeoutScanner claims timed-out games and delegates handling to service.HandleTimedOutGame.
func startTimeoutScanner(repo interface {
	ClaimTimedOutGameIDs(time.Time, int, time.Duration, string) ([]uint, error)
	GetGameByID(uint) (*game.Game, error)
	UpdateGame(*game.Game) error
}, actionTimeout time.Duration, workerID string) {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			ids, err := repo.ClaimTimedOutGameIDs(now, 20, 2*time.Minute, workerID)
			if err != nil {
				logging.Error("timeout scanner failed to list ids", err, nil)
				continue
			}
			if len(ids) == 0 {
				continue
			}
			// process each id sequentially (keeps DB safe under SQLite)
			for _, id := range ids {
				gg, err := repo.GetGameByID(id)
				if err != nil {
					continue
				}
				// delegate to service-level handler which encapsulates the
				// auto-rest and finish logic.
				_ = service.HandleTimedOutGame(repo.(interface {
					UpdateGame(*game.Game) error
					UpdateStatsOnGameEnd(*game.Game, string) error
					GetGameByID(uint) (*game.Game, error)
				}), gg, actionTimeout)
			}
		}
	}()
}
