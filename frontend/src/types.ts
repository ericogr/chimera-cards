export interface Animal {
  ID: number;
  name: string;
  pv: number;
  atq: number;
  def: number;
  agi: number;
  ene: number;
  vigor_cost?: number;
  skill_name: string;
  skill_cost: number;
  skill_description: string;
}

// Canonical animal names used across the frontend.
// Keeping names in a single enum reduces typos and eases maintenance.
export enum AnimalName {
  Lion = 'Lion',
  Bear = 'Bear',
  Cheetah = 'Cheetah',
  Eagle = 'Eagle',
  Rhino = 'Rhino',
  Turtle = 'Turtle',
  Gorilla = 'Gorilla',
  Wolf = 'Wolf',
  Octopus = 'Octopus',
  Raven = 'Raven',
}

export interface Hybrid {
  ID: number;
  name: string;
  generated_name?: string;
  base_animals: Animal[];
  selected_ability_animal_id?: number;
  base_pv: number;
  current_pv: number;
  base_atq: number;
  current_atq: number;
  base_def: number;
  current_def: number;
  base_agi: number;
  current_agi: number;
  base_ene: number;
  current_ene: number;
  base_vig?: number;
  current_vig?: number;
  is_active: boolean;
  is_defeated: boolean;
  // combat state (optional from backend)
  stunned_until_round?: number;
  last_action?: string;
}

export interface Player {
  ID: number;
  player_uuid: string;
  player_name: string;
  player_email?: string;
  has_created: boolean;
  has_submitted_action?: boolean;
  pending_action_type?: string;
  pending_action_animal_id?: number;
  hybrids: Hybrid[];
}

export interface Game {
  ID: number;
  name: string;
  description: string;
  private: boolean;
  join_code: string;
  players: Player[];
  current_turn: string;
  round_count: number;
  turn_number?: number;
  phase?: 'planning' | 'resolving' | 'resolved';
  status: string;
  winner?: string;
  message?: string;
  last_round_summary?: string;
  created_at: string;
}
