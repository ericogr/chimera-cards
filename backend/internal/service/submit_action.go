package service

import (
	"errors"

	"github.com/ericogr/quimera-cards/internal/engine"
	"github.com/ericogr/quimera-cards/internal/game"
)

var (
	ErrGameNotInProgress          = errors.New("game is not in progress")
	ErrActionsLocked              = errors.New("actions are locked; resolving current round")
	ErrPlayerNotInGame            = errors.New("player not in game")
	ErrNoActiveHybrid             = errors.New("no active hybrid")
	ErrHybridHasNoSelectedAbility = errors.New("hybrid has no selected ability")
	ErrAbilityMismatch            = errors.New("ability must match the hybrid's selected animal")
)

// SubmitAction stores a player's chosen action and resolves the round if both players submitted.
// Returns the updated game and a boolean indicating whether the round was resolved.
func SubmitAction(repo GameRepo, gameID uint, playerUUID string, actionType game.PendingActionType, animalID uint) (*game.Game, bool, error) {
	g, err := repo.GetGameByID(gameID)
	if err != nil || g == nil {
		return nil, false, ErrGameNotFound
	}
	if g.Status != "in_progress" {
		return nil, false, ErrGameNotInProgress
	}
	if g.Phase != "planning" {
		return nil, false, ErrActionsLocked
	}
	if len(g.Players) != 2 {
		return nil, false, errors.New("invalid player count")
	}

	var current *game.Player
	if g.Players[0].PlayerUUID == playerUUID {
		current = &g.Players[0]
	} else if g.Players[1].PlayerUUID == playerUUID {
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
		if active.SelectedAbilityAnimalID == nil {
			return nil, false, ErrHybridHasNoSelectedAbility
		}
		if animalID != 0 && animalID != *active.SelectedAbilityAnimalID {
			return nil, false, ErrAbilityMismatch
		}
		aid := *active.SelectedAbilityAnimalID
		current.PendingActionAnimalID = &aid
	} else {
		current.PendingActionAnimalID = nil
	}

	resolved := false
	if g.Players[0].HasSubmittedAction && g.Players[1].HasSubmittedAction {
		engine.ResolveRound(g)
		if g.Status == "finished" && !g.StatsCounted {
			_ = repo.UpdateStatsOnGameEnd(g, "")
			g.StatsCounted = true
		}
		resolved = true
	}

	if err := repo.UpdateGame(g); err != nil {
		return nil, resolved, err
	}

	return g, resolved, nil
}
