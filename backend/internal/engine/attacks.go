package engine

import (
	"math"
	"math/rand"
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

func (rc *roundContext) execSwiftPounce(plan *plannedAction, oppPlayer *game.Player) {
	atqEff := attackWithModifiers(plan.actor, rc.g.RoundCount)
	defEff := defenseWithModifiers(plan.target)
	ignored := false
	if plan.actor.AttackIgnoresDefenseThisRound {
		defEff = 0
		ignored = true
	}

	// Default calculation (backwards compatible) uses half of Agility.
	raw := atqEff - defEff

	// If the entity has a configured skill effect, apply its parameters.
	if plan.entity != nil {
		eff := plan.entity.SkillEffect

		// Partial defense ignore
		if eff.SwiftIgnoreDefensePercent > 0 {
			defEff = int(float64(defEff) * (1.0 - float64(eff.SwiftIgnoreDefensePercent)/100.0))
			if defEff < 0 {
				defEff = 0
			}
		}

		raw = atqEff - defEff
		if raw < 1 {
			raw = 1
		}

		divisor := eff.SwiftAddAgilityDivisor
		if divisor <= 0 {
			divisor = 2
		}

		dmg := raw + int(math.Floor(float64(plan.actor.CurrentAgility)/float64(divisor)))

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
		rc.add(plan.player.PlayerName + " SWIFT POUNCE — Calculation: Attack " + strconv.Itoa(atqEff) + ", Defense " + strconv.Itoa(defEff) + func() string {
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
		if len(ctxParts) > 0 {
			rc.add(oppPlayer.PlayerName + "'s " + hybridDisplayName(plan.target) + " takes " + strconv.Itoa(dmg) + " damage (" + strings.Join(ctxParts, ", ") + ")")
		} else {
			rc.add(oppPlayer.PlayerName + "'s " + hybridDisplayName(plan.target) + " takes " + strconv.Itoa(dmg) + " damage")
		}
		return
	}

	if raw < 1 {
		raw = 1
	}
	dmg := raw + int(math.Floor(float64(plan.actor.CurrentAgility)/2.0))
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
	rc.add(plan.player.PlayerName + " SWIFT POUNCE — Calculation: Attack " + strconv.Itoa(atqEff) + ", Defense " + strconv.Itoa(defEff) + func() string {
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
	if len(ctxParts) > 0 {
		rc.add(oppPlayer.PlayerName + "'s " + hybridDisplayName(plan.target) + " takes " + strconv.Itoa(dmg) + " damage (" + strings.Join(ctxParts, ", ") + ")")
	} else {
		rc.add(oppPlayer.PlayerName + "'s " + hybridDisplayName(plan.target) + " takes " + strconv.Itoa(dmg) + " damage")
	}
}

func (rc *roundContext) execCharge(plan *plannedAction, oppPlayer *game.Player) {
	atqEff := attackWithModifiers(plan.actor, rc.g.RoundCount)
	// Allow configuration to modify the extra attack applied by Charge.
	if plan.entity != nil && plan.entity.SkillEffect.ChargeExtraAttack != 0 {
		atqEff += plan.entity.SkillEffect.ChargeExtraAttack
	} else {
		atqEff += 5
	}
	defEff := defenseWithModifiers(plan.target)
	if plan.actor.AttackIgnoresDefenseThisRound {
		defEff = 0
	}
	raw := atqEff - defEff
	if raw < 1 {
		raw = 1
	}
	dmg := raw
	if plan.actor.AttackHalvedThisRound {
		dmg = int(math.Floor(float64(dmg) * 0.5))
		if dmg < 1 {
			dmg = 1
		}
	}
	if plan.target.VulnerableThisRound {
		dmg = int(math.Ceil(float64(dmg) * 1.25))
	}
	plan.target.CurrentHitPoints -= dmg
	recoilPercent := 0.2
	if plan.entity != nil && plan.entity.SkillEffect.ChargeRecoilPercent > 0 {
		recoilPercent = plan.entity.SkillEffect.ChargeRecoilPercent
	}
	recoil := int(float64(dmg) * recoilPercent)
	if recoil < 1 {
		recoil = 1
	}
	plan.actor.CurrentHitPoints -= recoil
	rc.add(plan.player.PlayerName + " CHARGE — Calculation: Attack " + strconv.Itoa(atqEff) + ", Defense " + strconv.Itoa(defEff) + "; final damage " + strconv.Itoa(dmg) + ", recoil to attacker " + strconv.Itoa(recoil))
}

func (rc *roundContext) execStun(plan *plannedAction, oppPlayer *game.Player) {
	atqEff := attackWithModifiers(plan.actor, rc.g.RoundCount)
	defEff := defenseWithModifiers(plan.target)
	if plan.actor.AttackIgnoresDefenseThisRound {
		defEff = 0
	}
	raw := atqEff - defEff
	if raw < 1 {
		raw = 1
	}
	dmg := raw
	if plan.actor.AttackHalvedThisRound {
		dmg = int(math.Floor(float64(dmg) * 0.5))
		if dmg < 1 {
			dmg = 1
		}
	}
	if plan.target.VulnerableThisRound {
		dmg = int(math.Ceil(float64(dmg) * 1.25))
	}
	plan.target.CurrentHitPoints -= dmg
	// Use configured stun chance and duration when available.
	stunned := false
	if plan.entity != nil && plan.entity.SkillEffect.StunChancePercent > 0 {
		stunned = rand.Intn(100) < plan.entity.SkillEffect.StunChancePercent
	} else {
		// Backwards compatible default (50% chance)
		stunned = rand.Intn(2) == 0
	}
	if stunned {
		dur := 1
		if plan.entity != nil && plan.entity.SkillEffect.StunDuration > 0 {
			dur = plan.entity.SkillEffect.StunDuration
		}
		plan.target.StunnedUntilRound = rc.g.RoundCount + dur
	}
	rc.add(plan.player.PlayerName + " STUN — Calculation: Attack " + strconv.Itoa(atqEff) + ", Defense " + strconv.Itoa(defEff) + "; final damage " + strconv.Itoa(dmg))
}
