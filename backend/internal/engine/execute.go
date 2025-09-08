package engine

import "github.com/ericogr/chimera-cards/internal/game"

// executePlans runs the prepared plans in order and records results in the context.
func (rc *roundContext) executePlans(plans []plannedAction) {
	opponentOf := func(p *game.Player) *game.Player {
		if p == &rc.g.Players[0] {
			return &rc.g.Players[1]
		}
		return &rc.g.Players[0]
	}

	for _, plan := range plans {
		if plan.actor.IsDefeated || plan.target.IsDefeated {
			continue
		}
		if plan.actor.CannotAttackUntilRound >= rc.g.RoundCount {
			continue
		}
		switch plan.action {
		case ActionBasicAttack:
			rc.execBasicAttack(&plan, opponentOf(plan.player))
		default:
			// All ability effects are applied as pre-effects; no separate
			// execution action is required.
		}

		if plan.target.CurrentHitPoints <= 0 && !plan.target.IsDefeated {
			plan.target.IsDefeated = true
			plan.target.IsActive = false
			rc.add(opponentOf(plan.player).PlayerName + "'s " + hybridDisplayName(plan.target) + " is defeated!")
		}
		if plan.actor.CurrentHitPoints <= 0 && !plan.actor.IsDefeated {
			plan.actor.IsDefeated = true
			plan.actor.IsActive = false
			rc.add(plan.player.PlayerName + "'s " + hybridDisplayName(plan.actor) + " is defeated!")
		}
	}
}
