import React, { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { apiFetch } from './api';
import { safeSetLocal } from './runtimeConfig';
import './Lobby.css';
import * as constants from './constants';
interface Player {
  ID: number;
  player_uuid: string;
  player_name: string;
  player_email?: string;
}

interface Game {
  ID: number;
  name: string;
  description: string;
  private: boolean;
  players: Player[];
  status: string;
  join_code: string;
  created_at: string; // Assuming date comes as a string
}

interface LobbyProps {
  user: {
    name?: string;
    email?: string;
    picture?: string;
  };
  onLogout: () => void;
}

const Lobby: React.FC<LobbyProps> = ({ user, onLogout }) => {
  const [games, setGames] = useState<Game[]>([]);
  
  const [error, setError] = useState<string | null>(null);
  const [leaderboard, setLeaderboard] = useState<Array<{ ID: number; PlayerName: string; Email: string; GamesPlayed: number; Wins: number; Resignations: number }>>([]);
  const [creatorStats, setCreatorStats] = useState<Record<string, { wins: number; resignations: number }>>({});
  const [joinCode, setJoinCode] = useState('');
  const [gameName, setGameName] = useState('');
  const [gameDescription, setGameDescription] = useState('');
  const [isPrivate, setIsPrivate] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const JOIN_CODE_LENGTH = 5;
  const navigate = useNavigate();
  const actingRef = useRef(false);

    useEffect(() => {
    const loadAvailableGames = async () => {
      try {
        const response = await apiFetch(constants.API_PUBLIC_GAMES);
        if (response.status === 401) { window.location.href = '/'; return; }
        if (!response.ok) {
          throw new Error('Failed to fetch games');
        }
        const data = await response.json();
        const available = (data || []).filter((g: Game) => g.status === 'waiting_for_players');
        setGames(available);
        // Trigger fetching creator stats for new creators
        const emails = new Set<string>();
        for (const g of available) {
          const creator = g.players?.[0]?.player_email;
          if (creator) emails.add(creator);
        }
        const missing = Array.from(emails).filter(e => !(e in creatorStats));
        if (missing.length) {
          const entries: [string, { wins: number; resignations: number }][] = [];
          await Promise.all(missing.map(async (e) => {
            try {
              const res = await apiFetch(`${constants.API_PLAYER_STATS}?email=${encodeURIComponent(e)}`);
              if (res.status === 401) { window.location.href = '/'; return; }
              if (res.ok) {
                const s = await res.json();
                entries.push([e, { wins: s.Wins ?? s.wins ?? 0, resignations: s.Resignations ?? s.resignations ?? 0 }]);
              }
            } catch {}
          }));
          if (entries.length) {
            setCreatorStats(prev => ({ ...prev, ...Object.fromEntries(entries) }));
          }
        }
      } catch (err) {
        setError('Could not load games. Please try again later.');
        console.error(err);
      }
    };

    // Ensure persistent player_uuid across sessions for stats aggregation
    try {
      let uuid = localStorage.getItem('player_uuid');
      if (!uuid) {
        uuid = window.crypto?.randomUUID ? window.crypto.randomUUID() : Math.random().toString(36).slice(2) + Date.now().toString(36);
        safeSetLocal('player_uuid', uuid);
      }
    } catch {}
    loadAvailableGames();
    // stats are shown in the shared header; Lobby does not fetch them locally
    const loadLeaderboard = async () => {
      try {
        const res = await apiFetch(constants.API_LEADERBOARD);
        if (res.ok) {
          const data = await res.json();
          setLeaderboard(Array.isArray(data) ? data : []);
        }
      } catch {}
    };

    loadLeaderboard();
    const intervalA = setInterval(loadAvailableGames, 5000);
    const intervalC = setInterval(loadLeaderboard, 10000);
    return () => { clearInterval(intervalA); clearInterval(intervalC); };
  }, [user?.email, creatorStats]);

  const createGame = async () => {
    if (!gameName.trim()) {
      alert('Please enter a name for the game.');
      return;
    }
    try {
      if (actingRef.current || submitting) return;
      actingRef.current = true;
      setSubmitting(true);
      const response = await apiFetch(constants.API_GAMES, {
        method: 'POST',
        headers: {
          [constants.HEADER_CONTENT_TYPE]: constants.CONTENT_TYPE_JSON,
        },
        body: JSON.stringify({
          name: gameName,
          description: gameDescription,
          private: isPrivate,
          player_name: user.name,
          player_email: user.email || '',
          player_uuid: localStorage.getItem('player_uuid') || '',
        }),
      });
      if (response.status === 401) { window.location.href = '/'; return; }
      if (!response.ok) {
        throw new Error('Failed to create game');
      }
      const newGameInfo = await response.json();
      try { if (user?.email) safeSetLocal('player_email', user.email); } catch {}
      safeSetLocal('game_id', newGameInfo.game_id);
      navigate(`/game/${newGameInfo.game_id}`);
    } catch (err) {
      alert('Error creating game. Please try again.');
      console.error(err);
    } finally { setSubmitting(false); actingRef.current = false; }
  };

  const joinGame = async (code?: string) => {
    const gameCodeToUse = code || joinCode;
    if (!gameCodeToUse.trim()) {
      alert('Please enter a game code.');
      return;
    }
    try {
      if (actingRef.current || submitting) return;
      actingRef.current = true;
      setSubmitting(true);
      const response = await apiFetch(constants.API_GAMES_JOIN, {
        method: 'POST',
        headers: {
          [constants.HEADER_CONTENT_TYPE]: constants.CONTENT_TYPE_JSON,
        },
        body: JSON.stringify({ 
          join_code: gameCodeToUse,
          player_name: user.name,
          player_email: user.email || '',
          player_uuid: localStorage.getItem('player_uuid') || '',
         }),
      });

      if (response.status === 401) { window.location.href = '/'; return; }
      if (response.ok) {
        const joinedGameInfo = await response.json();
        try { if (user?.email) safeSetLocal('player_email', user.email); } catch {}
        safeSetLocal('game_id', joinedGameInfo.game_id);
        navigate(`/game/${joinedGameInfo.game_id}`);
      } else {
        const errorData = await response.json();
        throw new Error(errorData.error || 'Failed to join game');
      }
    } catch (err: any) {
      alert(`Error joining game: ${err.message}`);
      console.error(err);
    } finally { setSubmitting(false); actingRef.current = false; }
  };

  return (
    <div>
      <main className="page-main">
        <h2>Game Lobby</h2>
        <div className="lobby-top">
              {/* Create Game Section */}
              <div>
                <h4>Create New Game</h4>
                <input
                  type="text"
                  placeholder="Game Name"
                  value={gameName}
                  onChange={(e) => setGameName(e.target.value)}
                  maxLength={32}
                  className="form-input"
                />
                <textarea
                  placeholder="Description"
                  value={gameDescription}
                  onChange={(e) => setGameDescription(e.target.value)}
                  maxLength={256}
                  className="form-textarea"
                />
            <div style={{ marginBottom: '20px' }}>
              <input 
                type="checkbox" 
                id="privateGame" 
                checked={isPrivate}
                onChange={(e) => setIsPrivate(e.target.checked)}
              />
              <label htmlFor="privateGame" style={{ marginLeft: '10px' }}>Private Game</label>
            </div>
            <button onClick={createGame} disabled={submitting || gameName.trim().length < 5} className="full-width">
              Create Game
            </button>
          </div>

          {/* Join Game Section */}
          <div>
            <h4>Join Existing Game</h4>
            <div className="join-row">
              <input
                type="text"
                placeholder="Enter game code"
                value={joinCode}
                onChange={(e) => setJoinCode(e.target.value.toUpperCase())}
                maxLength={JOIN_CODE_LENGTH}
                className="form-input"
                style={{ flexGrow: 1 }}
              />
              <button onClick={() => joinGame()} disabled={submitting || joinCode.trim().length !== JOIN_CODE_LENGTH}>
                Join Game
              </button>
            </div>
          </div>
        </div>

        <h4 className="mt-20">Available Public Games</h4>
        {error && <p style={{ color: 'red' }}>{error}</p>}
        <div className="table-wrap">
          <div className="game-list game-list-header">
            <div>Game</div>
            <div>Description</div>
            <div>Players</div>
            <div>Code</div>
            <div>Victory</div>
            <div>Resignations</div>
          </div>
          {games.length > 0 ? (
            games.map(game => (
              <div key={game.ID} className="game-list">
                <div className="game-name">
                  <button onClick={() => joinGame(game.join_code)} className="link-button" disabled={submitting}>
                    {game.name}
                  </button>
                </div>
                <div className="game-description">{game.description}</div>
                <div className="game-players">{game.players?.length || 0}/2</div>
                <div className="game-waiting">{game.join_code || '-'}</div>
                <div className="game-wins">{creatorStats[game.players?.[0]?.player_email || '']?.wins ?? 0}</div>
                <div className="game-resignations">{creatorStats[game.players?.[0]?.player_email || '']?.resignations ?? 0}</div>
              </div>
            ))
          ) : (
            <p>No public games available. Create one!</p>
          )}
        </div>
      </main>
      <section className="section-padding">
        <h4 className="mt-20">Top 10 Players</h4>
        <div className="table-wrap">
          <div className="rank-list rank-list-header">
            <div>#</div>
            <div>Name</div>
            <div>Wins</div>
            <div>Defeats</div>
            <div>Resignations</div>
          </div>
          {leaderboard.length > 0 ? (
            leaderboard.map((p, idx) => {
              const gp = (p as any).GamesPlayed ?? (p as any).games_played ?? 0;
              const wins = (p as any).Wins ?? (p as any).wins ?? 0;
              const resign = (p as any).Resignations ?? (p as any).resignations ?? 0;
              const defeatsRow = Math.max(0, gp - wins - resign);
              const name = (p as any).PlayerName ?? (p as any).player_name ?? 'Unknown';
              return (
                <div key={(p as any).ID ?? idx} className="rank-list">
                  <div>{idx + 1}</div>
                  <div>{name}</div>
                  <div>{wins}</div>
                  <div>{defeatsRow}</div>
                  <div>{resign}</div>
                </div>
              );
            })
          ) : (
            <p>No players yet. Play some games!</p>
          )}
        </div>
      </section>
    </div>
  );
};

export default Lobby;
