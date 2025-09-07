package engine

import "github.com/ericogr/chimera-cards/internal/game"

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
	rc.g.LastRoundSummary = rc.joinSummary()
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
