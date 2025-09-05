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

// Vigor cost is now stored on each Animal as `VigorCost` (configured in chimera_config.json).

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

// getChosen returns the animal object referenced by pid inside a hybrid.
func getChosen(h *game.Hybrid, pid *uint) *game.Animal {
	if pid == nil {
		return nil
	}
	for i := range h.BaseAnimals {
		if h.BaseAnimals[i].ID == *pid {
			return &h.BaseAnimals[i]
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
	animal *game.Animal
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
		var a *game.Animal
		if p.PendingActionType == game.PendingActionAbility {
			a = getChosen(self, p.PendingActionAnimalID)
		}
		switch p.PendingActionType {
		case game.PendingActionBasicAttack:
			plans = append(plans, plannedAction{player: p, actor: self, target: opp, action: ActionBasicAttack})
			self.LastAction = string(ActionBasicAttack)
		case game.PendingActionAbility:
			if a != nil {
				switch a.Name {
				case string(game.Cheetah):
					plans = append(plans, plannedAction{player: p, actor: self, target: opp, action: ActionSkillSwiftPounce, animal: a})
				case string(game.Rhino):
					plans = append(plans, plannedAction{player: p, actor: self, target: opp, action: ActionSkillCharge, animal: a})
				case string(game.Gorilla):
					plans = append(plans, plannedAction{player: p, actor: self, target: opp, action: ActionSkillStun, animal: a})
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
		ch := getChosen(self, p.PendingActionAnimalID)
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
		switch ch.Name {
		case string(game.Lion):
			opp.AttackDebuffPercent = 30
			opp.AttackDebuffUntilRound = rc.g.RoundCount
			self.LastAction = string(ActionSkillRoar)
			rc.add(p.PlayerName + " ABILITY — ROAR: -30% opponent Attack this round. Costs: Energy " + strconv.Itoa(rc.minInt(prevE, ch.SkillCost)) + ", Vigor " + strconv.Itoa(spentV) + rc.vulnerableTag(self))
		case string(game.Bear):
			self.AttackBuffPercent = 50
			self.AttackBuffUntilRound = rc.g.RoundCount
			self.SelfDefenseIgnoredUntilRound = rc.g.RoundCount
			self.AttackIgnoresDefenseThisRound = true
			self.LastAction = string(ActionSkillFrenzy)
			rc.add(p.PlayerName + " ABILITY — FRENZY: +50% Attack and ignores Defense this round. Costs: Energy " + strconv.Itoa(rc.minInt(prevE, ch.SkillCost)) + ", Vigor " + strconv.Itoa(spentV) + rc.vulnerableTag(self))
		case string(game.Eagle):
			self.PriorityNextRound = true
			self.LastAction = string(ActionSkillFlight)
			rc.add(p.PlayerName + " ABILITY — FLIGHT: gains priority next round. Costs: Energy " + strconv.Itoa(rc.minInt(prevE, ch.SkillCost)) + ", Vigor " + strconv.Itoa(spentV) + rc.vulnerableTag(self))
		case string(game.Turtle):
			self.DefenseBuffMultiplier = 3
			self.DefenseBuffUntilRound = rc.g.RoundCount
			self.CannotAttackUntilRound = rc.g.RoundCount
			self.LastAction = string(ActionSkillIronShell)
			rc.add(p.PlayerName + " ABILITY — IRON SHELL: Defense x3 this round (cannot attack). Costs: Energy " + strconv.Itoa(rc.minInt(prevE, ch.SkillCost)) + ", Vigor " + strconv.Itoa(spentV) + rc.vulnerableTag(self))
		case string(game.Wolf):
			self.CurrentEnergy += 4
			self.LastAction = string(ActionSkillPackTactics)
			rc.add(p.PlayerName + " ABILITY — PACK TACTICS: +4 Energy. Costs: Energy " + strconv.Itoa(rc.minInt(prevE, ch.SkillCost)) + ", Vigor " + strconv.Itoa(spentV) + rc.vulnerableTag(self))
		case string(game.Octopus):
			opp.AgilityDebuffPercent = 50
			opp.AgilityDebuffUntilRound = rc.g.RoundCount + 1
			self.LastAction = string(ActionSkillInk)
			rc.add(p.PlayerName + " ABILITY — INK CURTAIN: -50% opponent Agility for 2 rounds. Costs: Energy " + strconv.Itoa(rc.minInt(prevE, ch.SkillCost)) + ", Vigor " + strconv.Itoa(spentV) + rc.vulnerableTag(self))
		case string(game.Raven):
			self.LastAction = string(ActionSkillReveal)
			rc.add(p.PlayerName + " ABILITY — REVEAL: show opponent info. Costs: Energy " + strconv.Itoa(rc.minInt(prevE, ch.SkillCost)) + ", Vigor " + strconv.Itoa(spentV) + rc.vulnerableTag(self))
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
	raw := atqEff - defEff
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
	atqEff := attackWithModifiers(a.actor, rc.g.RoundCount) + 5
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
	recoil := int(float64(dmg) * 0.2)
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
	stunned := rand.Intn(2) == 0
	if stunned {
		a.target.StunnedUntilRound = rc.g.RoundCount + 1
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
			rc.g.Players[i].PendingActionAnimalID = nil
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
