import { useEffect, useState } from 'react';
import { apiFetch } from '../api';
import * as constants from '../constants';

export interface PlayerStats { GamesPlayed: number; Wins: number; Resignations: number }

export function usePlayerStats(email?: string | null) {
  const [stats, setStats] = useState<PlayerStats | null>(null);

  useEffect(() => {
    if (!email) return;
    let mounted = true;
    const load = async () => {
      try {
        const res = await apiFetch(`${constants.API_PLAYER_STATS}?email=${encodeURIComponent(email)}`);
        if (!res.ok) return;
        const data = await res.json();
        if (!mounted) return;
        setStats({ GamesPlayed: data.GamesPlayed ?? data.games_played ?? 0, Wins: data.Wins ?? data.wins ?? 0, Resignations: data.Resignations ?? data.resignations ?? 0 });
      } catch (e) {
        // ignore
      }
    };
    load();
    return () => { mounted = false; };
  }, [email]);

  return stats;
}

