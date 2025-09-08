package engine

import (
	"math"
	"strconv"
	"strings"

	"github.com/ericogr/chimera-cards/internal/game"
)

func (rc *roundContext) execBasicAttack(plan *plannedAction, oppPlayer *game.Player) {
	atqEff := attackWithModifiers(plan.actor, rc.g.RoundCount)
	defEff := defenseWithModifiers(plan.target)
	ignored := false
	if plan.actor.AttackIgnoresDefenseThisRound {
		defEff = 0
		ignored = true
	}
	raw := atqEff - defEff
	if raw < 1 {
		raw = 1
	}
	dmg := raw
	halved := false
	if plan.actor.AttackHalvedThisRound {
		dmg = int(math.Floor(float64(dmg) * 0.5))
		if dmg < 1 {
			dmg = 1
		}
		halved = true
	}
	appliedVuln := false
	if plan.target.VulnerableThisRound {
		dmg = int(math.Ceil(float64(dmg) * 1.25))
		appliedVuln = true
	}
	plan.target.CurrentHitPoints -= dmg
	rc.add(plan.player.PlayerName + " BASIC ATTACK — Calculation: Attack " + strconv.Itoa(atqEff) + ", Defense " + strconv.Itoa(defEff) + func() string {
		if ignored {
			return " (defense ignored)"
		}
		return ""
	}() + "; base damage " + strconv.Itoa(raw) + func() string {
		if halved {
			return " (halved due to 0 VIG)"
		}
		return ""
	}() + func() string {
		if appliedVuln {
			return "; +25% vs Vulnerable"
		}
		return ""
	}() + "; final damage " + strconv.Itoa(dmg))
	ctxParts := []string{}
	if plan.target.DefendStanceActive {
		ctxParts = append(ctxParts, "defend bonus")
	}
	if plan.target.DefenseBuffMultiplier > 0 && plan.target.DefenseBuffUntilRound >= rc.g.RoundCount {
		ctxParts = append(ctxParts, "Iron Shell")
	}
	if ignored {
		ctxParts = append(ctxParts, "ignored defense")
	}
	if appliedVuln {
		ctxParts = append(ctxParts, "+25% vs Vulnerable")
	}
	ctx := ""
	if len(ctxParts) > 0 {
		ctx = " (" + strings.Join(ctxParts, ", ") + ")"
	}
	rc.add(oppPlayer.PlayerName + "'s " + hybridDisplayName(plan.target) + " takes " + strconv.Itoa(dmg) + " damage" + ctx)
}
