import React from 'react';
import { useNavigate } from 'react-router-dom';
import SettingsMenu from './SettingsMenu';
import { usePlayerStats } from './hooks/usePlayerStats';
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
  const navigate = useNavigate();
  const stats = usePlayerStats(user?.email || null);
  const wins = stats?.Wins ?? 0;
  const gamesPlayed = stats?.GamesPlayed ?? 0;
  const resigns = stats?.Resignations ?? 0;
  const defeats = Math.max(0, gamesPlayed - wins - resigns);

  return (
    <header className="page-header shared-header">
      <div>
        <h3>{user?.name || 'Player'}</h3>
        <div className="header-stats">
          Wins: {wins} · Defeats: {defeats} · Resignations: {resigns}
        </div>
      </div>
      <div className="header-right">
        {user?.picture && <img src={user.picture} alt="Profile" className="header-avatar" />}
        <SettingsMenu onLogout={onLogout} onProfile={showProfileOption ? () => navigate('/profile') : undefined} />
      </div>
    </header>
  );
};

export default Header;
