package engine

import "github.com/ericogr/chimera-cards/internal/game"

// --- Modifier helpers --------------------------------------------------
func agilityWithModifiers(h *game.Hybrid, round int) int {
	agi := h.CurrentAgility
	if h.AgilityDebuffPercent > 0 && h.AgilityDebuffUntilRound >= round {
		agi = int(float64(agi) * (1.0 - float64(h.AgilityDebuffPercent)/100.0))
	}
	if agi < 0 {
		agi = 0
	}
	return agi
}

func defenseWithModifiers(h *game.Hybrid) int {
	d := h.CurrentDefense
	if h.DefenseBuffMultiplier > 0 && h.DefenseBuffUntilRound > 0 {
		d = d * h.DefenseBuffMultiplier
	}
	if h.DefendStanceActive {
		d = int(float64(d) * 1.5)
	}
	if d < 0 {
		d = 0
	}
	return d
}

func attackWithModifiers(h *game.Hybrid, round int) int {
	a := h.CurrentAttack
	if h.AttackDebuffPercent > 0 && h.AttackDebuffUntilRound >= round {
		a = int(float64(a) * (1.0 - float64(h.AttackDebuffPercent)/100.0))
	}
	if h.AttackBuffPercent > 0 && h.AttackBuffUntilRound >= round {
		a = int(float64(a) * (1.0 + float64(h.AttackBuffPercent)/100.0))
	}
	if a < 0 {
		a = 0
	}
	return a
}
