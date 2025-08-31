package service

import (
	"errors"

	"github.com/ericogr/quimera-cards/internal/constants"
	"github.com/ericogr/quimera-cards/internal/game"
	"github.com/ericogr/quimera-cards/internal/hybridimage"
	"github.com/ericogr/quimera-cards/internal/hybridname"
	"github.com/ericogr/quimera-cards/internal/logging"
	"github.com/ericogr/quimera-cards/internal/storage"
)

var (
	ErrPlayersNotReady      = errors.New("both players must create hybrids before starting")
	ErrEachPlayerTwoHybrids = errors.New("each player must have two hybrids")
)

// StartGame performs all server-side initialization when starting a game.
// It generates AI names for hybrids (or loads them from cache), initializes
// combat stats, and updates the game state. The provided game object is
// modified and persisted using the repository.
func StartGame(repo storage.Repository, g *game.Game) error {
	// Ensure both players created hybrids
	if len(g.Players) != 2 || !g.Players[0].HasCreated || !g.Players[1].HasCreated {
		return ErrPlayersNotReady
	}

	// Generate names and images (may call OpenAI if missing)
	if err := generateNamesAndImages(repo, g); err != nil {
		return err
	}

	// Initialize current stats and set first hybrid active
	for i := range g.Players {
		if len(g.Players[i].Hybrids) < 2 {
			return ErrEachPlayerTwoHybrids
		}
		for j := range g.Players[i].Hybrids {
			hbd := &g.Players[i].Hybrids[j]
			hbd.CurrentHitPoints = hbd.BaseHitPoints
			hbd.CurrentAttack = hbd.BaseAttack
			hbd.CurrentDefense = hbd.BaseDefense
			hbd.CurrentAgility = hbd.BaseAgility
			hbd.CurrentEnergy = hbd.BaseEnergy
			// Initialize VIG: base and current (fast, fair, simple: base=3)
			if hbd.BaseVIG == 0 {
				hbd.BaseVIG = 3
			}
			hbd.CurrentVIG = hbd.BaseVIG
			hbd.IsDefeated = false
			hbd.IsActive = (j == 0)
		}
	}

	// Prepare game state
	g.Status = "in_progress"
	g.RoundCount = 1
	g.TurnNumber = 1
	g.Phase = "planning"
	g.Message = "The game has started. Choose your actions."

	// Round start adjustments
	for i := range g.Players {
		g.Players[i].HasSubmittedAction = false
		g.Players[i].PendingActionType = game.PendingActionNone
		g.Players[i].PendingActionAnimalID = nil
		for j := range g.Players[i].Hybrids {
			if g.Players[i].Hybrids[j].IsActive && !g.Players[i].Hybrids[j].IsDefeated {
				g.Players[i].Hybrids[j].CurrentEnergy += 1
				g.Players[i].Hybrids[j].LastAction = ""
				g.Players[i].Hybrids[j].DefendStanceActive = false
				g.Players[i].Hybrids[j].DefenseBuffMultiplier = 0
				g.Players[i].Hybrids[j].DefenseBuffUntilRound = 0
				g.Players[i].Hybrids[j].AttackBuffPercent = 0
				g.Players[i].Hybrids[j].AttackBuffUntilRound = 0
				g.Players[i].Hybrids[j].AttackDebuffPercent = 0
				g.Players[i].Hybrids[j].AttackDebuffUntilRound = 0
			}
		}
	}

	// Persist the updated game
	if err := repo.UpdateGame(g); err != nil {
		return err
	}
	return nil
}

// generateNamesAndImages assigns GeneratedName for each hybrid (from cache
// or OpenAI) and ensures the hybrid image exists (generating via OpenAI
// if missing). Returns error if any generation fails.
func generateNamesAndImages(repo storage.Repository, g *game.Game) error {
	for i := range g.Players {
		for j := range g.Players[i].Hybrids {
			hbd := &g.Players[i].Hybrids[j]
			names := make([]string, len(hbd.BaseAnimals))
			ids := make([]uint, len(hbd.BaseAnimals))
			for k := range hbd.BaseAnimals {
				ids[k] = hbd.BaseAnimals[k].ID
				names[k] = hbd.BaseAnimals[k].Name
			}

			if gen, source, err := hybridname.GetOrCreateGeneratedName(repo, ids, names); err == nil && gen != "" {
				hbd.GeneratedName = gen
				logging.Info("game-start hybrid name assigned", logging.Fields{constants.LogFieldGameID: g.ID, constants.LogFieldPlayerIdx: i, constants.LogFieldHybridIdx: j, constants.LogFieldSource: source, constants.LogFieldName: gen})
			} else {
				// fallback: use the concatenated name but do not store it in DB
				hbd.GeneratedName = hbd.Name
				logging.Error("game-start hybrid name fallback", err, logging.Fields{constants.LogFieldGameID: g.ID, constants.LogFieldPlayerIdx: i, constants.LogFieldHybridIdx: j, constants.LogFieldName: hbd.Name})
			}

			if err := hybridimage.EnsureHybridImage(repo, names); err != nil {
				logging.Error("game-start failed to generate hybrid image", err, logging.Fields{constants.LogFieldGameID: g.ID, constants.LogFieldPlayerIdx: i, constants.LogFieldHybridIdx: j})
				return err
			}
		}
	}
	return nil
}
