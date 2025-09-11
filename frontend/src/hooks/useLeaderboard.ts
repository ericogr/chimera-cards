import { useEffect, useState } from 'react';
import { apiFetch } from '../api';
import * as constants from '../constants';

export function useLeaderboard(intervalMs = 10000) {
  const [leaderboard, setLeaderboard] = useState<any[]>([]);
  useEffect(() => {
    let mounted = true;
    const load = async () => {
      try {
        const res = await apiFetch(constants.API_LEADERBOARD);
        if (!res.ok) return;
        const data = await res.json();
        if (!mounted) return;
        setLeaderboard(Array.isArray(data) ? data : []);
      } catch {}
    };
    load();
    const id = window.setInterval(load, intervalMs);
    return () => { mounted = false; window.clearInterval(id); };
  }, [intervalMs]);
  return leaderboard;
}

