package service

import (
	"errors"
	"time"

	"github.com/ericogr/chimera-cards/internal/engine"
	"github.com/ericogr/chimera-cards/internal/game"
)

var (
	ErrGameNotInProgress          = errors.New("game is not in progress")
	ErrActionsLocked              = errors.New("actions are locked; resolving current round")
	ErrPlayerNotInGame            = errors.New("player not in game")
	ErrNoActiveHybrid             = errors.New("no active hybrid")
	ErrHybridHasNoSelectedAbility = errors.New("hybrid has no selected ability")
	ErrAbilityMismatch            = errors.New("ability must match the hybrid's selected entity")
)

// SubmitAction stores a player's chosen action and resolves the round if both players submitted.
// Returns the updated game and a boolean indicating whether the round was resolved.
func SubmitAction(repo GameRepo, gameID uint, playerEmail string, actionType game.PendingActionType, entityID uint, actionTimeout time.Duration) (*game.Game, bool, error) {
	g, err := repo.GetGameByID(gameID)
	if err != nil || g == nil {
		return nil, false, ErrGameNotFound
	}
	if g.Status != game.StatusInProgress {
		return nil, false, ErrGameNotInProgress
	}
	if g.Phase != game.PhasePlanning {
		return nil, false, ErrActionsLocked
	}
	if len(g.Players) != 2 {
		return nil, false, errors.New("invalid player count")
	}

	var current *game.Player
    if (g.Players[0].PlayerEmail) == playerEmail {
        current = &g.Players[0]
    } else if (g.Players[1].PlayerEmail) == playerEmail {
        current = &g.Players[1]
    } else {
        return nil, false, ErrPlayerNotInGame
    }

	var active *game.Hybrid
	for i := range current.Hybrids {
		if current.Hybrids[i].IsActive && !current.Hybrids[i].IsDefeated {
			active = &current.Hybrids[i]
			break
		}
	}
	if active == nil {
		return nil, false, ErrNoActiveHybrid
	}

	current.HasSubmittedAction = true
	current.PendingActionType = actionType
	if actionType == game.PendingActionAbility {
		if active.SelectedAbilityEntityID == nil {
			return nil, false, ErrHybridHasNoSelectedAbility
		}
		if entityID != 0 && entityID != *active.SelectedAbilityEntityID {
			return nil, false, ErrAbilityMismatch
		}
		aid := *active.SelectedAbilityEntityID
		current.PendingActionEntityID = &aid
	} else {
		current.PendingActionEntityID = nil
	}

	resolved := false
	if g.Players[0].HasSubmittedAction && g.Players[1].HasSubmittedAction {
		engine.ResolveRound(g)
		// If the match continues, reset the action deadline for the next round;
		// otherwise mark stats as counted so no further updates occur.
		if g.Status == game.StatusFinished {
			if !g.StatsCounted {
				// keep existing behavior for normal finishes
				_ = repo.UpdateStatsOnGameEnd(g, "")
				g.StatsCounted = true
			}
		} else {
			// New planning phase started; reset deadline
			g.ActionDeadline = time.Now().Add(actionTimeout)
		}
		resolved = true
	}

	if err := repo.UpdateGame(g); err != nil {
		return nil, resolved, err
	}

	return g, resolved, nil
}
