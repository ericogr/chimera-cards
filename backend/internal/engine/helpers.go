package engine

import "github.com/ericogr/chimera-cards/internal/game"

// findActiveHybrid returns the active, non-defeated hybrid for a player.
func findActiveHybrid(p *game.Player) *game.Hybrid {
	for i := range p.Hybrids {
		if p.Hybrids[i].IsActive && !p.Hybrids[i].IsDefeated {
			return &p.Hybrids[i]
		}
	}
	return nil
}

// getChosen returns the entity object referenced by pid inside a hybrid.
func getChosen(h *game.Hybrid, pid *uint) *game.Entity {
	if pid == nil {
		return nil
	}
	for i := range h.BaseEntities {
		if h.BaseEntities[i].ID == *pid {
			return &h.BaseEntities[i]
		}
	}
	return nil
}

// hybridDisplayName returns the generated name when available, otherwise
// falls back to the derived combination name.
func hybridDisplayName(h *game.Hybrid) string {
	if h == nil {
		return ""
	}
	if h.GeneratedName != "" {
		return h.GeneratedName
	}
	return h.Name
}
