import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '../api';
import * as constants from '../constants';

export interface PublicGame {
  ID: number;
  name: string;
  description: string;
  private: boolean;
  players: any[];
  status: string;
  join_code: string;
  created_at: string;
}

export function usePublicGames(intervalMs = 5000) {
  const [games, setGames] = useState<PublicGame[]>([]);
  const [error, setError] = useState<string | null>(null);

  const fetchGames = useCallback(async () => {
    try {
      const res = await apiFetch(constants.API_PUBLIC_GAMES);
      if (res.status === 401) { window.location.href = '/'; return; }
      if (!res.ok) throw new Error('Failed to fetch games');
      const data = await res.json();
      const available = (data || []).filter((g: PublicGame) => g.status === 'waiting_for_players');
      setGames(available);
      setError(null);
    } catch (e: any) {
      setError(e.message || 'Failed to load public games');
    }
  }, []);

  useEffect(() => {
    fetchGames();
    const id = window.setInterval(fetchGames, intervalMs);
    return () => window.clearInterval(id);
  }, [fetchGames, intervalMs]);

  return { games, error, refresh: fetchGames } as const;
}

