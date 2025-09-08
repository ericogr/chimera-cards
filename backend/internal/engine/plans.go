package engine

import (
	"math/rand"
	"sort"

	"github.com/ericogr/chimera-cards/internal/game"
)

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

	mapPlan := func(player *game.Player, self, opp *game.Hybrid) {
		switch player.PendingActionType {
		case game.PendingActionBasicAttack:
			plans = append(plans, plannedAction{player: player, actor: self, target: opp, action: ActionBasicAttack})
		// Abilities are applied during pre-effect resolution (applyAbilityPreEffects)
		// and do not create separate execution plans anymore.
		case game.PendingActionAbility:
			// no execution plan; pre-effects already applied
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
