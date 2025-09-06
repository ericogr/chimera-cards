package engine

import (
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"

	"github.com/ericogr/quimera-cards/internal/game"
)

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

// Vigor cost is now stored on each Entity as `VigorCost` (configured in chimera_config.json).

// --- Round context and helpers ----------------------------------------
type roundContext struct {
	g       *game.Game
	summary []string
}

func newRoundContext(g *game.Game) *roundContext {
	return &roundContext{g: g, summary: make([]string, 0, 16)}
}

func (rc *roundContext) add(msg string) { rc.summary = append(rc.summary, msg) }

func (rc *roundContext) vulnerableTag(h *game.Hybrid) string {
	if h.VulnerableThisRound {
		return " — becomes Vulnerable (+25% damage this round)"
	}
	return ""
}

func (rc *roundContext) minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

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

// --- Planned action model ---------------------------------------------
type plannedAction struct {
	player *game.Player
	actor  *game.Hybrid
	target *game.Hybrid
	action ActionKind
	entity *game.Entity
}

type ActionKind string

const (
	ActionNone        ActionKind = ""
	ActionBasicAttack ActionKind = "basic_attack"
	ActionDefend      ActionKind = "defend"
	ActionAbility     ActionKind = "ability"
	ActionRest        ActionKind = "rest"

	// Ability-specific actions (skill:<name>) used by the resolver.
	ActionSkillSwiftPounce ActionKind = "skill:swift_pounce"
	ActionSkillCharge      ActionKind = "skill:charge"
	ActionSkillStun        ActionKind = "skill:stun"
	ActionSkillRoar        ActionKind = "skill:roar"
	ActionSkillFrenzy      ActionKind = "skill:frenzy"
	ActionSkillFlight      ActionKind = "skill:flight"
	ActionSkillIronShell   ActionKind = "skill:iron_shell"
	ActionSkillPackTactics ActionKind = "skill:pack_tactics"
	ActionSkillInk         ActionKind = "skill:ink"
	ActionSkillReveal      ActionKind = "skill:reveal"
)

// buildPlans converts player pending actions into executable plannedAction list.
func (rc *roundContext) buildPlans(p1, p2 *game.Player, h1, h2 *game.Hybrid) []plannedAction {
	plans := make([]plannedAction, 0, 4)
	mapPlan := func(p *game.Player, self, opp *game.Hybrid) {
		var a *game.Entity
		if p.PendingActionType == game.PendingActionAbility {
			a = getChosen(self, p.PendingActionEntityID)
		}
		switch p.PendingActionType {
		case game.PendingActionBasicAttack:
			plans = append(plans, plannedAction{player: p, actor: self, target: opp, action: ActionBasicAttack})
			// LastAction is set in applyPreEffects so we don't duplicate that logic here.
		case game.PendingActionAbility:
			if a != nil {
				// If the ability requires an execution step (e.g., performs
				// direct damage), the configuration should mark it with
				// SkillEffect.ExecutesPlan = true. Use the configured
				// skill_key as the action identifier so new entities can be
				// added in config without touching code.
				if a.SkillEffect.ExecutesPlan {
					act := ActionAbility
					if a.SkillKey != "" {
						act = ActionKind(a.SkillKey)
					}
					plans = append(plans, plannedAction{player: p, actor: self, target: opp, action: act, entity: a})
				}
			}
		}
	}
	mapPlan(p1, h1, h2)
	mapPlan(p2, h2, h1)

	// sort by agility (desc), tie -> random
	sort.SliceStable(plans, func(i, j int) bool {
		ai := agilityWithModifiers(plans[i].actor, rc.g.RoundCount)
		aj := agilityWithModifiers(plans[j].actor, rc.g.RoundCount)
		if ai == aj {
			return rand.Intn(2) == 0
		}
		return ai > aj
	})
	return plans
}

// applyPreEffects handles costs and immediate effects of chosen actions
// (e.g., VIG cost, ENE cost, temporary buffs/debuffs).
func (rc *roundContext) applyPreEffects(p *game.Player, self, opp *game.Hybrid) {
	// reset round flags
	self.AttackHalvedThisRound = false
	self.VulnerableThisRound = false
	self.AttackIgnoresDefenseThisRound = false
	switch p.PendingActionType {
	case game.PendingActionDefend:
		prevV := self.CurrentVIG
		if self.CurrentVIG > 0 {
			self.CurrentVIG -= 1
			self.DefendStanceActive = true
		} else {
			self.DefendStanceActive = false
		}
		self.LastAction = string(ActionDefend)
		if prevV > 0 {
			rc.add(p.PlayerName + " DEFEND: spent 1 VIG (+50% Defense this round)")
		} else {
			rc.add(p.PlayerName + " DEFEND: 0 VIG — no defense bonus")
		}
	case game.PendingActionAbility:
		ch := getChosen(self, p.PendingActionEntityID)
		if ch == nil {
			return
		}
		prevE := self.CurrentEnergy
		if self.CurrentEnergy >= ch.SkillCost {
			self.CurrentEnergy -= ch.SkillCost
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
		// Apply ability effects from configuration rather than a hard-coded
		// per-entity switch. This lets new entities be added via the
		// chimera_config.json file without touching engine code.
		eff := ch.SkillEffect

		// Record a stable last-action key when available (used for UI/logs).
		if ch.SkillKey != "" {
			self.LastAction = ch.SkillKey
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
			rc.add(p.PlayerName + " ABILITY — " + ch.SkillName + ": -" + strconv.Itoa(eff.OpponentAttackDebuffPercent) + "% opponent Attack for " + strconv.Itoa(dur) + " round(s). Costs: Energy " + strconv.Itoa(rc.minInt(prevE, ch.SkillCost)) + ", Vigor " + strconv.Itoa(spentV) + rc.vulnerableTag(self))
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
			rc.add(p.PlayerName + " ABILITY — " + ch.SkillName + ": +" + strconv.Itoa(eff.AttackBuffPercent) + "% Attack" + func() string {
				if eff.AttackIgnoresDefense {
					return " and ignores Defense"
				}
				return ""
			}() + ". Costs: Energy " + strconv.Itoa(rc.minInt(prevE, ch.SkillCost)) + ", Vigor " + strconv.Itoa(spentV) + rc.vulnerableTag(self))
		}

		// Priority
		if eff.PriorityNextRound {
			self.PriorityNextRound = true
			rc.add(p.PlayerName + " ABILITY — " + ch.SkillName + ": gains priority next round. Costs: Energy " + strconv.Itoa(rc.minInt(prevE, ch.SkillCost)) + ", Vigor " + strconv.Itoa(spentV) + rc.vulnerableTag(self))
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
			rc.add(p.PlayerName + " ABILITY — " + ch.SkillName + ": Defense x" + strconv.Itoa(eff.DefenseBuffMultiplier) + " for " + strconv.Itoa(dur) + " round(s)" + func() string {
				if eff.CannotAttack {
					return " (cannot attack)"
				}
				return ""
			}() + ". Costs: Energy " + strconv.Itoa(rc.minInt(prevE, ch.SkillCost)) + ", Vigor " + strconv.Itoa(spentV) + rc.vulnerableTag(self))
		}

		// Restore energy
		if eff.RestoreEnergy > 0 {
			self.CurrentEnergy += eff.RestoreEnergy
			rc.add(p.PlayerName + " ABILITY — " + ch.SkillName + ": +" + strconv.Itoa(eff.RestoreEnergy) + " Energy. Costs: Energy " + strconv.Itoa(rc.minInt(prevE, ch.SkillCost)) + ", Vigor " + strconv.Itoa(spentV) + rc.vulnerableTag(self))
		}

		// Opponent agility debuff
		if eff.OpponentAgilityDebuffPercent > 0 {
			dur := eff.OpponentAgilityDebuffDuration
			if dur <= 0 {
				dur = 1
			}
			opp.AgilityDebuffPercent = eff.OpponentAgilityDebuffPercent
			opp.AgilityDebuffUntilRound = rc.g.RoundCount + dur - 1
			rc.add(p.PlayerName + " ABILITY — " + ch.SkillName + ": -" + strconv.Itoa(eff.OpponentAgilityDebuffPercent) + "% opponent Agility for " + strconv.Itoa(dur) + " round(s). Costs: Energy " + strconv.Itoa(rc.minInt(prevE, ch.SkillCost)) + ", Vigor " + strconv.Itoa(spentV) + rc.vulnerableTag(self))
		}

		// Reveal informational ability
		if eff.Reveal {
			rc.add(p.PlayerName + " ABILITY — " + ch.SkillName + ": reveal opponent info. Costs: Energy " + strconv.Itoa(rc.minInt(prevE, ch.SkillCost)) + ", Vigor " + strconv.Itoa(spentV) + rc.vulnerableTag(self))
		}
	case game.PendingActionBasicAttack:
		prevV := self.CurrentVIG
		if self.CurrentVIG > 0 {
			self.CurrentVIG -= 1
			self.AttackHalvedThisRound = false
		} else {
			self.AttackHalvedThisRound = true
		}
		self.LastAction = string(ActionBasicAttack)
		if prevV > 0 {
			rc.add(p.PlayerName + " BASIC ATTACK: spent 1 VIG")
		} else {
			rc.add(p.PlayerName + " BASIC ATTACK: 0 VIG — damage will be halved")
		}
	case game.PendingActionRest:
		self.CurrentVIG += 2
		if self.BaseVIG > 0 && self.CurrentVIG > self.BaseVIG {
			self.CurrentVIG = self.BaseVIG
		}
		self.CurrentEnergy += 2
		self.LastAction = string(ActionRest)
		rc.add(p.PlayerName + " REST: +2 VIG, +2 ENE (VIG capped at base)")
	}
}

// executePlans runs the prepared plans in order and records results in the context.
func (rc *roundContext) executePlans(plans []plannedAction) {
	opponentOf := func(p *game.Player) *game.Player {
		if p == &rc.g.Players[0] {
			return &rc.g.Players[1]
		}
		return &rc.g.Players[0]
	}

	for _, act := range plans {
		if act.actor.IsDefeated || act.target.IsDefeated {
			continue
		}
		if act.actor.CannotAttackUntilRound >= rc.g.RoundCount {
			continue
		}
		switch act.action {
		case ActionBasicAttack:
			rc.execBasicAttack(&act, opponentOf(act.player))
		case ActionSkillSwiftPounce:
			rc.execSwiftPounce(&act, opponentOf(act.player))
		case ActionSkillCharge:
			rc.execCharge(&act, opponentOf(act.player))
		case ActionSkillStun:
			rc.execStun(&act, opponentOf(act.player))
		}

		if act.target.CurrentHitPoints <= 0 && !act.target.IsDefeated {
			act.target.IsDefeated = true
			act.target.IsActive = false
			rc.add(opponentOf(act.player).PlayerName + "'s " + act.target.Name + " is defeated!")
		}
		if act.actor.CurrentHitPoints <= 0 && !act.actor.IsDefeated {
			act.actor.IsDefeated = true
			act.actor.IsActive = false
			rc.add(act.player.PlayerName + "'s " + act.actor.Name + " is defeated!")
		}
	}
}

func (rc *roundContext) execBasicAttack(a *plannedAction, oppPlayer *game.Player) {
	atqEff := attackWithModifiers(a.actor, rc.g.RoundCount)
	defEff := defenseWithModifiers(a.target)
	ignored := false
	if a.actor.AttackIgnoresDefenseThisRound {
		defEff = 0
		ignored = true
	}
	raw := atqEff - defEff
	if raw < 1 {
		raw = 1
	}
	dmg := raw
	halved := false
	if a.actor.AttackHalvedThisRound {
		dmg = int(math.Floor(float64(dmg) * 0.5))
		if dmg < 1 {
			dmg = 1
		}
		halved = true
	}
	appliedVuln := false
	if a.target.VulnerableThisRound {
		dmg = int(math.Ceil(float64(dmg) * 1.25))
		appliedVuln = true
	}
	a.target.CurrentHitPoints -= dmg
	rc.add(a.player.PlayerName + " BASIC ATTACK — Calculation: Attack " + strconv.Itoa(atqEff) + ", Defense " + strconv.Itoa(defEff) + func() string {
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
	if a.target.DefendStanceActive {
		ctxParts = append(ctxParts, "defend bonus")
	}
	if a.target.DefenseBuffMultiplier > 0 && a.target.DefenseBuffUntilRound >= rc.g.RoundCount {
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
	rc.add(oppPlayer.PlayerName + "'s " + a.target.Name + " takes " + strconv.Itoa(dmg) + " damage" + ctx)
}

func (rc *roundContext) execSwiftPounce(a *plannedAction, oppPlayer *game.Player) {
	atqEff := attackWithModifiers(a.actor, rc.g.RoundCount)
	defEff := defenseWithModifiers(a.target)
	ignored := false
	if a.actor.AttackIgnoresDefenseThisRound {
		defEff = 0
		ignored = true
	}

	// Default calculation (backwards compatible) uses half of Agility.
	raw := atqEff - defEff

	// If the entity has a configured skill effect, apply its parameters.
	if a.entity != nil {
		eff := a.entity.SkillEffect

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

		dmg := raw + int(math.Floor(float64(a.actor.CurrentAgility)/float64(divisor)))

		halved := false
		if a.actor.AttackHalvedThisRound {
			dmg = int(math.Floor(float64(dmg) * 0.5))
			if dmg < 1 {
				dmg = 1
			}
			halved = true
		}

		appliedVuln := false
		if a.target.VulnerableThisRound {
			dmg = int(math.Ceil(float64(dmg) * 1.25))
			appliedVuln = true
		}

		a.target.CurrentHitPoints -= dmg
		rc.add(a.player.PlayerName + " SWIFT POUNCE — Calculation: Attack " + strconv.Itoa(atqEff) + ", Defense " + strconv.Itoa(defEff) + func() string {
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
		if a.target.DefendStanceActive {
			ctxParts = append(ctxParts, "defend bonus")
		}
		if a.target.DefenseBuffMultiplier > 0 && a.target.DefenseBuffUntilRound >= rc.g.RoundCount {
			ctxParts = append(ctxParts, "Iron Shell")
		}
		if ignored {
			ctxParts = append(ctxParts, "ignored defense")
		}
		if appliedVuln {
			ctxParts = append(ctxParts, "+25% vs Vulnerable")
		}
		if len(ctxParts) > 0 {
			rc.add(oppPlayer.PlayerName + "'s " + a.target.Name + " takes " + strconv.Itoa(dmg) + " damage (" + strings.Join(ctxParts, ", ") + ")")
		} else {
			rc.add(oppPlayer.PlayerName + "'s " + a.target.Name + " takes " + strconv.Itoa(dmg) + " damage")
		}
		return
	}

	if raw < 1 {
		raw = 1
	}
	dmg := raw + int(math.Floor(float64(a.actor.CurrentAgility)/2.0))
	halved := false
	if a.actor.AttackHalvedThisRound {
		dmg = int(math.Floor(float64(dmg) * 0.5))
		if dmg < 1 {
			dmg = 1
		}
		halved = true
	}
	appliedVuln := false
	if a.target.VulnerableThisRound {
		dmg = int(math.Ceil(float64(dmg) * 1.25))
		appliedVuln = true
	}
	a.target.CurrentHitPoints -= dmg
	rc.add(a.player.PlayerName + " SWIFT POUNCE — Calculation: Attack " + strconv.Itoa(atqEff) + ", Defense " + strconv.Itoa(defEff) + func() string {
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
	if a.target.DefendStanceActive {
		ctxParts = append(ctxParts, "defend bonus")
	}
	if a.target.DefenseBuffMultiplier > 0 && a.target.DefenseBuffUntilRound >= rc.g.RoundCount {
		ctxParts = append(ctxParts, "Iron Shell")
	}
	if ignored {
		ctxParts = append(ctxParts, "ignored defense")
	}
	if appliedVuln {
		ctxParts = append(ctxParts, "+25% vs Vulnerable")
	}
	if len(ctxParts) > 0 {
		rc.add(oppPlayer.PlayerName + "'s " + a.target.Name + " takes " + strconv.Itoa(dmg) + " damage (" + strings.Join(ctxParts, ", ") + ")")
	} else {
		rc.add(oppPlayer.PlayerName + "'s " + a.target.Name + " takes " + strconv.Itoa(dmg) + " damage")
	}
}

func (rc *roundContext) execCharge(a *plannedAction, oppPlayer *game.Player) {
	atqEff := attackWithModifiers(a.actor, rc.g.RoundCount)
	// Allow configuration to modify the extra attack applied by Charge.
	if a.entity != nil && a.entity.SkillEffect.ChargeExtraAttack != 0 {
		atqEff += a.entity.SkillEffect.ChargeExtraAttack
	} else {
		atqEff += 5
	}
	defEff := defenseWithModifiers(a.target)
	if a.actor.AttackIgnoresDefenseThisRound {
		defEff = 0
	}
	raw := atqEff - defEff
	if raw < 1 {
		raw = 1
	}
	dmg := raw
	if a.actor.AttackHalvedThisRound {
		dmg = int(math.Floor(float64(dmg) * 0.5))
		if dmg < 1 {
			dmg = 1
		}
	}
	if a.target.VulnerableThisRound {
		dmg = int(math.Ceil(float64(dmg) * 1.25))
	}
	a.target.CurrentHitPoints -= dmg
	recoilPercent := 0.2
	if a.entity != nil && a.entity.SkillEffect.ChargeRecoilPercent > 0 {
		recoilPercent = a.entity.SkillEffect.ChargeRecoilPercent
	}
	recoil := int(float64(dmg) * recoilPercent)
	if recoil < 1 {
		recoil = 1
	}
	a.actor.CurrentHitPoints -= recoil
	rc.add(a.player.PlayerName + " CHARGE — Calculation: Attack " + strconv.Itoa(atqEff) + ", Defense " + strconv.Itoa(defEff) + "; final damage " + strconv.Itoa(dmg) + ", recoil to attacker " + strconv.Itoa(recoil))
}

func (rc *roundContext) execStun(a *plannedAction, oppPlayer *game.Player) {
	atqEff := attackWithModifiers(a.actor, rc.g.RoundCount)
	defEff := defenseWithModifiers(a.target)
	if a.actor.AttackIgnoresDefenseThisRound {
		defEff = 0
	}
	raw := atqEff - defEff
	if raw < 1 {
		raw = 1
	}
	dmg := raw
	if a.actor.AttackHalvedThisRound {
		dmg = int(math.Floor(float64(dmg) * 0.5))
		if dmg < 1 {
			dmg = 1
		}
	}
	if a.target.VulnerableThisRound {
		dmg = int(math.Ceil(float64(dmg) * 1.25))
	}
	a.target.CurrentHitPoints -= dmg
	// Use configured stun chance and duration when available.
	stunned := false
	if a.entity != nil && a.entity.SkillEffect.StunChancePercent > 0 {
		stunned = rand.Intn(100) < a.entity.SkillEffect.StunChancePercent
	} else {
		// Backwards compatible default (50% chance)
		stunned = rand.Intn(2) == 0
	}
	if stunned {
		dur := 1
		if a.entity != nil && a.entity.SkillEffect.StunDuration > 0 {
			dur = a.entity.SkillEffect.StunDuration
		}
		a.target.StunnedUntilRound = rc.g.RoundCount + dur
	}
	rc.add(a.player.PlayerName + " STUN — Calculation: Attack " + strconv.Itoa(atqEff) + ", Defense " + strconv.Itoa(defEff) + "; final damage " + strconv.Itoa(dmg))
}

// bringReserve promotes the next non-defeated hybrid to active if needed.
func (rc *roundContext) bringReserve(p *game.Player) {
	if len(p.Hybrids) < 2 {
		return
	}
	active := false
	for i := range p.Hybrids {
		if p.Hybrids[i].IsActive && !p.Hybrids[i].IsDefeated {
			active = true
		}
	}
	if !active {
		for i := range p.Hybrids {
			if !p.Hybrids[i].IsDefeated && !p.Hybrids[i].IsActive {
				h := &p.Hybrids[i]
				h.IsActive = true
				h.CurrentHitPoints = h.BaseHitPoints
				h.CurrentAttack = h.BaseAttack
				h.CurrentDefense = h.BaseDefense
				h.CurrentAgility = h.BaseAgility
				h.CurrentEnergy = h.BaseEnergy
				break
			}
		}
	}
}

// finalizeRound evaluates victory conditions and prepares next round or resolves the match.
func (rc *roundContext) finalizeRound() {
	p1 := &rc.g.Players[0]
	p2 := &rc.g.Players[1]
	cntDefeated := func(p *game.Player) int {
		d := 0
		for i := range p.Hybrids {
			if p.Hybrids[i].IsDefeated {
				d++
			}
		}
		return d
	}
	if cntDefeated(p1) == 2 {
		rc.g.Status = "finished"
		rc.g.Winner = p2.PlayerName
		rc.g.Message = "Victory for player " + p2.PlayerName
	} else if cntDefeated(p2) == 2 {
		rc.g.Status = "finished"
		rc.g.Winner = p1.PlayerName
		rc.g.Message = "Victory for player " + p1.PlayerName
	}

	// next round or resolved
	rc.g.LastRoundSummary = strings.Join(rc.summary, "\n")
	if rc.g.Status == "in_progress" {
		rc.g.RoundCount++
		rc.g.TurnNumber = 1
		for i := range rc.g.Players {
			rc.g.Players[i].HasSubmittedAction = false
			rc.g.Players[i].PendingActionType = game.PendingActionNone
			rc.g.Players[i].PendingActionEntityID = nil
			for j := range rc.g.Players[i].Hybrids {
				if rc.g.Players[i].Hybrids[j].IsActive && !rc.g.Players[i].Hybrids[j].IsDefeated {
					// +1 ENE at round start
					rc.g.Players[i].Hybrids[j].CurrentEnergy += 1
					// Fatigue schedule: Round 3:-1, Round 4:-2, Round 5+:-3
					if rc.g.RoundCount >= 3 {
						dec := 1
						if rc.g.RoundCount >= 5 {
							dec = 3
						} else if rc.g.RoundCount >= 4 {
							dec = 2
						}
						rc.g.Players[i].Hybrids[j].CurrentDefense -= dec
						if rc.g.Players[i].Hybrids[j].CurrentDefense < 0 {
							rc.g.Players[i].Hybrids[j].CurrentDefense = 0
						}
					}
					rc.g.Players[i].Hybrids[j].DefendStanceActive = false
					rc.g.Players[i].Hybrids[j].DefenseBuffMultiplier = 0
					rc.g.Players[i].Hybrids[j].DefenseBuffUntilRound = 0
					rc.g.Players[i].Hybrids[j].AttackBuffPercent = 0
					rc.g.Players[i].Hybrids[j].AttackBuffUntilRound = 0
					rc.g.Players[i].Hybrids[j].AttackDebuffPercent = 0
					rc.g.Players[i].Hybrids[j].AttackDebuffUntilRound = 0
					rc.g.Players[i].Hybrids[j].AttackHalvedThisRound = false
					rc.g.Players[i].Hybrids[j].VulnerableThisRound = false
					rc.g.Players[i].Hybrids[j].AttackIgnoresDefenseThisRound = false
				}
			}
		}
		rc.g.Phase = "planning"
		rc.g.Message = "New round. Choose your actions."
	} else {
		rc.g.Phase = "resolved"
	}
}

// ResolveRound is the main entry point for resolving a round. It orchestrates
// pre-effects, execution of planned actions and round finalization.
func ResolveRound(g *game.Game) {
	if len(g.Players) != 2 {
		return
	}
	// begin
	g.Phase = "resolving"
	rc := newRoundContext(g)

	p1 := &g.Players[0]
	p2 := &g.Players[1]
	h1 := findActiveHybrid(p1)
	h2 := findActiveHybrid(p2)
	if h1 == nil || h2 == nil {
		return
	}

	// Stun checks
	if h1.StunnedUntilRound >= g.RoundCount {
		p1.PendingActionType = game.PendingActionSkip
		h1.LastAction = "stunned"
	}
	if h2.StunnedUntilRound >= g.RoundCount {
		p2.PendingActionType = game.PendingActionSkip
		h2.LastAction = "stunned"
	}

	// Pre-effects and costs
	rc.applyPreEffects(p1, h1, h2)
	rc.applyPreEffects(p2, h2, h1)

	// Build & execute plans
	plans := rc.buildPlans(p1, p2, h1, h2)
	rc.executePlans(plans)

	// Bring reserves and finalize
	rc.bringReserve(p1)
	rc.bringReserve(p2)
	rc.finalizeRound()
}
