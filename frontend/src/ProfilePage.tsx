import React, { useEffect, useState } from 'react';
import * as constants from './constants';
import { apiFetch } from './api';
import { safeSetLocal } from './runtimeConfig';
import { useNavigate } from 'react-router-dom';

interface Props {
  user: { name?: string; email?: string; picture?: string } | null;
  onLogout: () => void;
}

const ProfilePage: React.FC<Props> = ({ user, onLogout }) => {
  const navigate = useNavigate();
  const [name, setName] = useState(user?.name || '');
  const [loading, setLoading] = useState(false);
  const [stats, setStats] = useState<{ GamesPlayed: number; Wins: number; Resignations: number } | null>(null);
  const [message, setMessage] = useState<string | null>(null);

  useEffect(() => {
    const fetchStats = async () => {
      if (!user?.email) return;
      try {
        const res = await apiFetch(`${constants.API_PLAYER_STATS}?email=${encodeURIComponent(user.email)}`);
        if (!res.ok) {
          return;
        }
        const data = await res.json();
        setStats({ GamesPlayed: data.GamesPlayed ?? data.games_played ?? 0, Wins: data.Wins ?? data.wins ?? 0, Resignations: data.Resignations ?? data.resignations ?? 0 });
      } catch (e) {
        console.error('Failed to load stats', e);
      }
    };
    fetchStats();
  }, [user?.email]);

  const saveName = async () => {
    if (!user?.email) return setMessage('Missing email');
    setLoading(true);
    setMessage(null);
    try {
      const res = await apiFetch(constants.API_PLAYER_STATS, {
        method: 'POST',
        headers: { [constants.HEADER_CONTENT_TYPE]: constants.CONTENT_TYPE_JSON },
        body: JSON.stringify({ name }),
      });
      if (!res.ok) {
        const txt = await res.text();
        throw new Error(txt || 'Failed to save');
      }
      // Update local copy
      const stored = localStorage.getItem('user');
      if (stored) {
        try {
          const parsed = JSON.parse(stored);
          parsed.name = name;
          safeSetLocal('user', JSON.stringify(parsed));
        } catch {}
      }
      setMessage('Saved');
    } catch (e: any) {
      setMessage(`Error: ${e.message}`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="page-main">
      <section className="content-section--narrow">
        <div className="row-between">
          <h2 className="no-margin">User Profile</h2>
        </div>
        <div style={{ marginBottom: 12 }}>
          <label style={{ display: 'block', marginBottom: 6 }}>Display Name</label>
          <input value={name} onChange={e => setName(e.target.value)} className="form-input" />
        </div>
        <div style={{ marginBottom: 12, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <button onClick={saveName} disabled={loading}>{loading ? 'Saving…' : 'Save'}</button>
            {message && <span className="ml-12">{message}</span>}
          </div>
          <div>
            <button onClick={() => navigate('/')}>Back</button>
          </div>
        </div>

        <h3>Statistics</h3>
        <table className="table-light">
          <thead>
            <tr>
              <th>Wins</th>
              <th>Defeats</th>
              <th>Resignations</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>{stats ? stats.Wins : '-'}</td>
              <td>{stats ? Math.max(0, (stats.GamesPlayed ?? 0) - (stats.Wins ?? 0) - (stats.Resignations ?? 0)) : '-'}</td>
              <td>{stats ? stats.Resignations : '-'}</td>
            </tr>
          </tbody>
        </table>
      </section>
    </div>
  );
};

export default ProfilePage;
