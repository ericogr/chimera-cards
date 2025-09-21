# Chimera: The Battle of Beasts

## Core Concept

Players combine the essence of 2–3 distinct entities to forge two unique hybrid creatures. Each hybrid inherits strengths and weaknesses from its progenitors’ traits. The goal is to use foresight, stamina management, and tactical bluffing to defeat the opponent’s two hybrids in tense, simultaneous-action combat.

This system keeps battles sharp and decisive — hybrids can unleash powerful abilities, but dwindling stamina and creeping fatigue ensure that every round pushes the fight toward a dramatic finish.

## 1. Entity Attributes

To keep the game focused and streamlined, each entity is defined by a set of core attributes. Every base entity has a fixed distribution of points across its stats, ensuring that no creature is strictly better — only specialized in a unique role.

| Name       | Abbr. | Description                                                                |
| ---------- | ----- | -------------------------------------------------------------------------- |
| Hit Points | HP    | Your hybrid’s life total. At 0 HP, the hybrid is defeated.                 |
| Attack     | ATK   | The raw damage output of your hybrid.                                      |
| Defense    | DEF   | Reduces incoming damage from basic and special attacks.                    |
| Agility    | AGI   | Determines priority; can also enable critical strikes or evasions.         |
| Energy     | ENE   | Resource used to activate Special Abilities. Regenerates slowly.           |
| Vigor      | VIG   | Stamina and endurance. Actions consume VIG; when depleted, you weaken.     |

### Energy vs. Vigor

- Energy fuels abilities and regenerates slowly. Energy is intentionally scarce: each entity contributes little ENE (0–1) by role, and a hybrid's base ENE is clamped between 1 and 3 at creation.
- Vigor is the wear-and-tear of combat — if it runs out, your hybrid’s effectiveness collapses.

## 2. The 10 Fundamental Entities

| Entity     | HP | ATK | DEF | AGI | ENE | Special Ability (Energy Cost)                                                              |
| ---------- | -- | --- | --- | --- | --- | ------------------------------------------------------------------------------------------ |
| Lion       | 4  | 8   | 4   | 5   | 2   | Commanding Roar (3 ENE): Reduces opponent’s ATK by 30% for one round.                     |
| Bear       | 6  | 7   | 5   | 2   | 3   | Frenzy (4 ENE): Increases own ATK by 50% this round, ignoring DEF.                        |
| Cheetah    | 3  | 5   | 2   | 10  | 4   | Swift Pounce (3 ENE): +30% Attack and ignores 40% of opponent's DEF for one round.        |
| Eagle      | 2  | 6   | 2   | 9   | 5   | Strategic Flight (2 ENE): +20% Attack next round.                                          |
| Rhinoceros | 7  | 6   | 7   | 1   | 2   | Relentless Charge (4 ENE): +40% Attack for one round.                                     |
| Turtle     | 8  | 1   | 9   | 1   | 4   | Iron Shell (3 ENE): Triples DEF for one round but cannot attack.                          |
| Gorilla    | 6  | 7   | 6   | 2   | 2   | Stunning Blow (5 ENE): +30% Attack this round.                                              |
| Wolf       | 4  | 5   | 4   | 6   | 5   | Pack Tactics (2 ENE): Restores 4 Energy points.                                           |
| Octopus    | 5  | 2   | 5   | 4   | 8   | Ink Curtain (3 ENE): Reduces opponent’s AGI by 50% for 2 rounds.                          |
| Raven      | 2  | 3   | 3   | 7   | 9   | Cunning Analysis (2 ENE): Restores 2 Energy to your hybrid.                                |

## 3. Game Phases

## Skill Effects

Each entity's Special Ability is described in two ways: a human-friendly `skill.name`
and `skill.description`, and a machine-readable `skill.effect` object in the server
configuration. The `skill.effect` object contains a small set of optional parameters
that the engine uses to apply buffs, debuffs and other mechanical behaviour when the
ability is used. This makes it easy to add or tweak abilities without changing code.


Common `skill.effect` parameters (simple explanations):

- `opponent_attack_debuff_percent` / `opponent_attack_debuff_duration`: reduce the
  opponent's Attack by a percentage for N rounds.
- `opponent_agility_debuff_percent` / `opponent_agility_debuff_duration`: reduce the
  opponent's Agility for N rounds.
- `attack_buff_percent` / `attack_buff_duration`: increase the user's Attack for N rounds.
- `attack_ignores_defense` / `attack_ignores_defense_duration`: ignore opponent's
  Defense for the configured duration.
- `defense_buff_multiplier` / `defense_buff_duration`: multiply user's Defense for N rounds.
- `cannot_attack` / `cannot_attack_duration`: prevent the user from attacking for N rounds.
- `restore_energy`: immediately restore ENE to the user.

Example 1 — Commanding Roar (Lion): reduce opponent ATK by 30% for one round

```json
"skill": {
  "name": "Commanding Roar",
  "effect": {
    "opponent_attack_debuff_percent": 30,
    "opponent_attack_debuff_duration": 1
  }
}
```

Example 2 — Overpower / Charge (Rhino): replace execution-specific charge options
with a simple attack buff representation

```json
"skill": {
  "name": "Relentless Charge",
  "effect": {
    "attack_buff_percent": 40,
    "attack_buff_duration": 1
  }
}
```

See `backend/chimera_config.json` for concrete examples used in the game.

### Phase 1: Creation

- Selection: Each player secretly chooses 2–3 different entities to create Hybrid 1, then another 2–3 different entities to create Hybrid 2. The same entity cannot appear in both hybrids of the same player.
- Calculation: A hybrid’s attributes are the sum of its chosen entities. HP and stats add directly. For Special Abilities, the player must select exactly ONE of the hybrid’s entities to define the hybrid’s unique ability for the whole match.
- Revelation: Both players reveal Hybrid 1 and place it in the arena. Hybrid 2 remains hidden in reserve until needed.

### Phase 2: Combat

- The battle is fought in rounds until one player defeats both enemy hybrids.

## 4. Combat Mechanics

### Round Start

- Each hybrid regenerates +1 Energy (base hybrid Energy is low: 1–3).
- If a hybrid chooses Rest, it also gains +2 Energy in addition to +2 Vigor.
- Vigor does not regenerate passively.

### Action timeout behaviour

- Each planning phase may have a server-enforced action deadline (per-round timeout).
- If the deadline expires and exactly one player failed to submit an action, the server
  automatically treats that player's action as `Rest` and resolves the round immediately
  using the same engine rules as a normal simultaneous resolution (the round may end the
  match or continue as usual).
- If the deadline expires and both players failed to submit actions, the match ends due
  to inactivity: the match is marked finished with no winner and no stats are awarded.

This rule applies only to the combat planning phase. The hybrid creation flow is
unaffected by the action-timeout rule.

### Simultaneous Action Choice

Both players secretly choose one action, then reveal them at the same time. Actions are resolved simultaneously with AGI as a tiebreaker when relevant.

#### Possible Actions

- Basic Attack (cost: 1 VIG): Damage = ATK – opponent’s DEF (minimum 1). If VIG = 0, the attack deals only half damage.
- Defend (cost: 1 VIG): Increases DEF by 50% for this round. If VIG = 0, the defense does not apply and the hybrid takes full damage.
- Special Ability (cost: ENE + variable VIG): Uses the single ability chosen during creation for that hybrid. Abilities cost Energy plus 1–3 VIG depending on strength. If VIG = 0, the ability still works but leaves the hybrid vulnerable (takes +25% incoming damage this round).
- Rest (no cost): Regain +2 VIG and +2 ENE. Very risky if the opponent attacks.

#### Resolution Examples

- Attack vs Attack: both hybrids trade damage.
- Attack vs Defense: damage reduced, but both lose Vigor.
- Attack vs Rest: defender takes full damage, but regains resources.
- Defense vs Defense: nothing happens, but both lose 1 Vigor.
- Special vs Any: ability resolves, but still affected by opponent’s action.

### Battle Fatigue (Global Rule)

Starting from Round 3, prolonged combat wears down all hybrids:

- Round 3: all hybrids lose −1 DEF permanently.
- Round 4: −2 DEF permanently.
- Round 5 onward: −3 DEF permanently.

DEF cannot go below 0. This ensures the battle escalates quickly and prevents stalemates.

## 5. End of Battle

- When a hybrid’s HP reaches 0, it is defeated.
- The controlling player immediately brings Hybrid 2 into the arena.
- Victory is declared when one player defeats both enemy hybrids.

Note on resignations: if a player chooses to resign/end the match, the act is
recorded as a resignation for that player (used for stats). No victory is
awarded to the opponent when a resignation occurs.

## 6. Balancing Risks and Solutions

- The Untouchable Speedster (max AGI): Balanced by low HP/DEF. Fatigue ensures fragile hybrids collapse in prolonged fights.
- The Impenetrable Wall: High defense eventually crumbles under fatigue, and many abilities bypass defense altogether.
- Ability Abuse: Energy regeneration is slower, and high VIG costs prevent spamming. Resting becomes a risky but necessary gamble.
- Turtle Strategy (constant defense): Vigor depletion punishes repeated defending, making it unsustainable.

## 7. Why It’s Fun

- Every round is a psychological duel: “Will they attack, defend, or rest?”
- Vigor creates wear-and-tear tension where spamming the same move leads to collapse.
- Energy forces harder choices: resting gives powerful recovery, but leaves you wide open.
- Fatigue escalates battles rapidly, ensuring they climax within a handful of rounds.
- The simultaneous reveal mechanic keeps both players on edge — every decision feels critical.
