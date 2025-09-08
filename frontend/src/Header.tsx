import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import SettingsMenu from './SettingsMenu';
import { apiFetch } from './api';
import * as constants from './constants';
import './Header.css';

interface User {
  name?: string;
  email?: string;
  picture?: string;
}

interface Props {
  user: User | null;
  onLogout: () => void;
  showProfileOption?: boolean;
}

const Header: React.FC<Props> = ({ user, onLogout, showProfileOption }) => {
  const [stats, setStats] = useState<{ GamesPlayed: number; Wins: number; Resignations: number } | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const email = user?.email;
        if (!email) return;
        const res = await apiFetch(`${constants.API_PLAYER_STATS}?email=${encodeURIComponent(email)}`);
        if (!res.ok) return;
        const data = await res.json();
        setStats({ GamesPlayed: data.GamesPlayed ?? data.games_played ?? 0, Wins: data.Wins ?? data.wins ?? 0, Resignations: data.Resignations ?? data.resignations ?? 0 });
      } catch (e) {
        // ignore
      }
    };
    fetchStats();
  }, [user?.email]);

  const wins = stats?.Wins ?? 0;
  const gamesPlayed = stats?.GamesPlayed ?? 0;
  const resigns = stats?.Resignations ?? 0;
  const defeats = Math.max(0, gamesPlayed - wins - resigns);

  return (
    <header className="page-header shared-header">
      <div>
        <h3 style={{ margin: 0 }}>{user?.name || 'Player'}</h3>
        <div style={{ fontSize: 12, color: '#bbb', marginTop: 4 }}>
          Wins: {wins} · Defeats: {defeats} · Resignations: {resigns}
        </div>
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        {user?.picture && <img src={user.picture} alt="Profile" style={{ borderRadius: '50%', height: '40px' }} />}
        <SettingsMenu onLogout={onLogout} onProfile={showProfileOption ? () => navigate('/profile') : undefined} />
      </div>
    </header>
  );
};

export default Header;

