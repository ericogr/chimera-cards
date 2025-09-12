import React, { useEffect, useState } from 'react';
import { Button } from './ui';
import * as constants from './constants';
import { apiFetch } from './api';
import { safeSetLocal } from './runtimeConfig';
import { useNavigate } from 'react-router-dom';

interface Props {
  user: { name?: string; email?: string; picture?: string } | null;
  onLogout: () => void;
  onUserUpdate?: (u: { name?: string; email?: string; picture?: string } | null) => void;
}

const ProfilePage: React.FC<Props> = ({ user, onLogout, onUserUpdate }) => {
  const navigate = useNavigate();
  const [name, setName] = useState(user?.name || '');
  const [loading, setLoading] = useState(false);
  const [stats, setStats] = useState<{ GamesPlayed: number; Wins: number; Resignations: number } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [nameError, setNameError] = useState<string | null>(null);

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

  const nameRegex = /^[\p{L}\p{M}\p{N}.'\- ]{4,40}$/u;

  const saveName = async () => {
    setError(null);
    setNameError(null);
    if (!user?.email) return setError('Missing email');
    const trimmed = name.trim();
    if (!nameRegex.test(trimmed)) {
      setNameError('Invalid name');
      return;
    }
    setLoading(true);
    try {
      const res = await apiFetch(constants.API_PLAYER_STATS, {
        method: 'POST',
        headers: { [constants.HEADER_CONTENT_TYPE]: constants.CONTENT_TYPE_JSON },
        body: JSON.stringify({ name: trimmed }),
      });
      if (!res.ok) {
        const txt = await res.text();
        throw new Error(txt || 'Failed to save');
      }
      // Update local storage and notify parent to update header
      try {
        const stored = localStorage.getItem('user');
        let parsed: any = stored ? JSON.parse(stored) : { email: user.email };
        parsed.name = trimmed;
        safeSetLocal('user', JSON.stringify(parsed));
        if (typeof onUserUpdate === 'function') {
          onUserUpdate(parsed);
        }
      } catch {}
      // Redirect to home
      navigate('/');
    } catch (e: any) {
      setError(`Error: ${e.message}`);
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
        <div className="mb-12">
          <label className="form-label">Display Name</label>
          <input value={name} onChange={e => setName(e.target.value)} className="form-input" />
        </div>
        <div className="row-between mb-12">
          <div className="row-center">
            <Button onClick={saveName} disabled={loading || !nameRegex.test(name.trim())}>{loading ? 'Saving…' : 'Save'}</Button>
            {nameError && <span className="ml-12 error-message">{nameError}</span>}
            {error && <span className="ml-12 error-message">{error}</span>}
          </div>
          <div>
            <Button variant="ghost" onClick={() => navigate('/')}>Back</Button>
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
