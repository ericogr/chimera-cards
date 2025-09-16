import { useCallback, useEffect, useState } from 'react';
import { apiFetch } from '../api';
import * as constants from '../constants';
import { Game } from '../types';

export function useGame(gameId?: string | null, intervalMs = 3000) {
  const [game, setGame] = useState<Game | null>(null);
  const [error, setError] = useState<string | null>(null);

  const fetchGame = useCallback(async () => {
    if (!gameId) return;
    try {
      const response = await apiFetch(`${constants.API_GAMES}/${gameId}`);
      if (!response.ok) {
        setError('Game not found or an error occurred');
        return;
      }
      const data = await response.json();
      setGame(data);
      setError(null);
    } catch (e) {
      setError('Could not load game data.');
    }
  }, [gameId]);

  useEffect(() => {
    if (!gameId) return;
    let mounted = true;
    // initial + polling
    (async () => {
      try {
        const response = await apiFetch(`${constants.API_GAMES}/${gameId}`);
        if (!mounted) return;
        if (!response.ok) {
          setError('Game not found or an error occurred');
          return;
        }
        const data = await response.json();
        setGame(data);
        setError(null);
      } catch (e) {
        if (!mounted) return;
        setError('Could not load game data.');
      }
    })();

    const id = window.setInterval(() => {
      fetchGame();
    }, intervalMs);

    return () => {
      mounted = false;
      window.clearInterval(id);
    };
  }, [fetchGame, gameId, intervalMs]);

  return { game, error, refresh: fetchGame } as const;
}
