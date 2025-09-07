package engine

import (
	"strconv"

	"github.com/ericogr/chimera-cards/internal/game"
)

// applyPreEffects handles costs and immediate effects of chosen actions
// (e.g., VIG cost, ENE cost, temporary buffs/debuffs).
func (rc *roundContext) applyPreEffects(player *game.Player, self, opp *game.Hybrid) {
	// reset round flags
	self.AttackHalvedThisRound = false
	self.VulnerableThisRound = false
	self.AttackIgnoresDefenseThisRound = false

	switch player.PendingActionType {
	case game.PendingActionDefend:
		rc.applyDefendPreEffects(player, self)
	case game.PendingActionAbility:
		rc.applyAbilityPreEffects(player, self, opp)
	case game.PendingActionBasicAttack:
		rc.applyBasicAttackPreEffects(player, self)
	case game.PendingActionRest:
		rc.applyRestPreEffects(player, self)
	}
}

func (rc *roundContext) applyDefendPreEffects(player *game.Player, self *game.Hybrid) {
	prevV := self.CurrentVIG
	if self.CurrentVIG > 0 {
		self.CurrentVIG -= 1
		self.DefendStanceActive = true
	} else {
		self.DefendStanceActive = false
	}
	self.LastAction = string(ActionDefend)
	if prevV > 0 {
		rc.add(player.PlayerName + " DEFEND: spent 1 VIG (+50% Defense this round)")
	} else {
		rc.add(player.PlayerName + " DEFEND: 0 VIG — no defense bonus")
	}
}

func (rc *roundContext) applyAbilityPreEffects(player *game.Player, self, opp *game.Hybrid) {
	ch := getChosen(self, player.PendingActionEntityID)
	if ch == nil {
		return
	}
	prevE := self.CurrentEnergy
	if self.CurrentEnergy >= ch.Skill.Cost {
		self.CurrentEnergy -= ch.Skill.Cost
	}
	vigCost := ch.VigorCost
	prevV := self.CurrentVIG
	spentV := 0
	if self.CurrentVIG >= vigCost {
		self.CurrentVIG -= vigCost
		spentV = vigCost
	} else {
		spentV = prevV
		self.CurrentVIG = 0
		self.VulnerableThisRound = true
	}
	eff := ch.Skill.Effect

	// Record a stable last-action key when available (used for UI/logs).
	if ch.Skill.Key != "" {
		self.LastAction = ch.Skill.Key
	} else {
		self.LastAction = string(ActionAbility)
	}

	// Opponent attack debuff
	if eff.OpponentAttackDebuffPercent > 0 {
		dur := eff.OpponentAttackDebuffDuration
		if dur <= 0 {
			dur = 1
		}
		opp.AttackDebuffPercent = eff.OpponentAttackDebuffPercent
		opp.AttackDebuffUntilRound = rc.g.RoundCount + dur - 1
		rc.add(player.PlayerName + " ABILITY — " + ch.Skill.Name + ": -" + strconv.Itoa(eff.OpponentAttackDebuffPercent) + "% opponent Attack for " + strconv.Itoa(dur) + " round(s). Costs: Energy " + strconv.Itoa(rc.minInt(prevE, ch.Skill.Cost)) + ", Vigor " + strconv.Itoa(spentV) + rc.vulnerableTag(self))
	}

	// Self attack buff + optionally ignore defense
	if eff.AttackBuffPercent > 0 {
		dur := eff.AttackBuffDuration
		if dur <= 0 {
			dur = 1
		}
		self.AttackBuffPercent = eff.AttackBuffPercent
		self.AttackBuffUntilRound = rc.g.RoundCount + dur - 1
		if eff.AttackIgnoresDefense {
			self.AttackIgnoresDefenseThisRound = true
			if eff.AttackIgnoresDefenseDuration > 0 {
				self.SelfDefenseIgnoredUntilRound = rc.g.RoundCount + eff.AttackIgnoresDefenseDuration - 1
			} else {
				self.SelfDefenseIgnoredUntilRound = rc.g.RoundCount
			}
		}
		rc.add(player.PlayerName + " ABILITY — " + ch.Skill.Name + ": +" + strconv.Itoa(eff.AttackBuffPercent) + "% Attack" + func() string {
			if eff.AttackIgnoresDefense {
				return " and ignores Defense"
			}
			return ""
		}() + ". Costs: Energy " + strconv.Itoa(rc.minInt(prevE, ch.Skill.Cost)) + ", Vigor " + strconv.Itoa(spentV) + rc.vulnerableTag(self))
	}


	// Defense buff / cannot attack
	if eff.DefenseBuffMultiplier > 0 {
		dur := eff.DefenseBuffDuration
		if dur <= 0 {
			dur = 1
		}
		self.DefenseBuffMultiplier = eff.DefenseBuffMultiplier
		self.DefenseBuffUntilRound = rc.g.RoundCount + dur - 1
		if eff.CannotAttack {
			self.CannotAttackUntilRound = rc.g.RoundCount + eff.CannotAttackDuration - 1
		}
		rc.add(player.PlayerName + " ABILITY — " + ch.Skill.Name + ": Defense x" + strconv.Itoa(eff.DefenseBuffMultiplier) + " for " + strconv.Itoa(dur) + " round(s)" + func() string {
			if eff.CannotAttack {
				return " (cannot attack)"
			}
			return ""
		}() + ". Costs: Energy " + strconv.Itoa(rc.minInt(prevE, ch.Skill.Cost)) + ", Vigor " + strconv.Itoa(spentV) + rc.vulnerableTag(self))
	}

	// Restore energy
	if eff.RestoreEnergy > 0 {
		self.CurrentEnergy += eff.RestoreEnergy
		rc.add(player.PlayerName + " ABILITY — " + ch.Skill.Name + ": +" + strconv.Itoa(eff.RestoreEnergy) + " Energy. Costs: Energy " + strconv.Itoa(rc.minInt(prevE, ch.Skill.Cost)) + ", Vigor " + strconv.Itoa(spentV) + rc.vulnerableTag(self))
	}

	// Opponent agility debuff
	if eff.OpponentAgilityDebuffPercent > 0 {
		dur := eff.OpponentAgilityDebuffDuration
		if dur <= 0 {
			dur = 1
		}
		opp.AgilityDebuffPercent = eff.OpponentAgilityDebuffPercent
		opp.AgilityDebuffUntilRound = rc.g.RoundCount + dur - 1
		rc.add(player.PlayerName + " ABILITY — " + ch.Skill.Name + ": -" + strconv.Itoa(eff.OpponentAgilityDebuffPercent) + "% opponent Agility for " + strconv.Itoa(dur) + " round(s). Costs: Energy " + strconv.Itoa(rc.minInt(prevE, ch.Skill.Cost)) + ", Vigor " + strconv.Itoa(spentV) + rc.vulnerableTag(self))
	}

    // Note: priority/reveal mechanics were removed; abilities should use
    // the remaining structured parameters (buffs/debuffs, restore, etc.).
}

func (rc *roundContext) applyBasicAttackPreEffects(player *game.Player, self *game.Hybrid) {
	prevV := self.CurrentVIG
	if self.CurrentVIG > 0 {
		self.CurrentVIG -= 1
		self.AttackHalvedThisRound = false
	} else {
		self.AttackHalvedThisRound = true
	}
	self.LastAction = string(ActionBasicAttack)
	if prevV > 0 {
		rc.add(player.PlayerName + " BASIC ATTACK: spent 1 VIG")
	} else {
		rc.add(player.PlayerName + " BASIC ATTACK: 0 VIG — damage will be halved")
	}
}

func (rc *roundContext) applyRestPreEffects(player *game.Player, self *game.Hybrid) {
	self.CurrentVIG += 2
	if self.BaseVIG > 0 && self.CurrentVIG > self.BaseVIG {
		self.CurrentVIG = self.BaseVIG
	}
	self.CurrentEnergy += 2
	self.LastAction = string(ActionRest)
	rc.add(player.PlayerName + " REST: +2 VIG, +2 ENE (VIG capped at base)")
}
