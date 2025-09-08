import React, { useState, useEffect, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import HybridCreation from './HybridCreation';
import SettingsMenu from './SettingsMenu';
import { Game, Player } from './types';
import { apiFetch } from './api';
import * as constants from './constants';

const GameRoom: React.FC = () => {
  const { gameId } = useParams<{ gameId: string }>();
  const navigate = useNavigate();
  const [game, setGame] = useState<Game | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const actingRef = useRef(false);
  const hasLeftRef = useRef(false);
  const toBoardRef = useRef(false);

  const currentPlayerUUID = localStorage.getItem('player_uuid');

  useEffect(() => {
    const loadGame = async () => {
      try {
        const response = await apiFetch(`${constants.API_GAMES}/${gameId}`);
        if (!response.ok) {
          throw new Error('Game not found or an error occurred');
        }
        const data = await response.json();
        setGame(data);
        // Keep the Start button disabled while the game is transitioning
        // to the in-progress state. Also redirect immediately when the
        // game becomes active.
        if (data.status === 'starting') {
          setSubmitting(true);
        } else if (data.status === 'in_progress') {
            setSubmitting(false);
            toBoardRef.current = true;
            navigate(`/game/${gameId}/board`);
        } else if (data.status === 'error') {
            // Stop the disabled state so players can retry/cancel.
            setSubmitting(false);
        } else {
            // Any other state (e.g. waiting_for_players) should enable
            // the Start button when appropriate.
            setSubmitting(false);
        }

      } catch (err) {
        setError('Could not load game data.');
        console.error(err);
      }
    };

    loadGame();
    const interval = setInterval(loadGame, 3000);
    return () => clearInterval(interval);
  }, [gameId, navigate]);

  const canAutoLeaveRef = useRef(false);
  useEffect(() => {
    const currentPlayer: Player | undefined = game?.players?.find(p => p.player_uuid === currentPlayerUUID);
    canAutoLeaveRef.current = game?.status === 'waiting_for_players' && !!currentPlayer;
  }, [game, currentPlayerUUID]);

  useEffect(() => {
    const leaveIfEligible = () => {
      if (hasLeftRef.current) return;
      if (!canAutoLeaveRef.current) return;
      if (toBoardRef.current) return;
      hasLeftRef.current = true;
      try {
        const body = JSON.stringify({ player_uuid: currentPlayerUUID });
        fetch(`${constants.API_GAMES}/${gameId}/leave`, {
          method: 'POST',
          headers: { [constants.HEADER_CONTENT_TYPE]: constants.CONTENT_TYPE_JSON },
          body,
          keepalive: true,
          credentials: 'include',
        }).catch(() => {
          try {
            const blob = new Blob([body], { type: 'application/json' });
            // @ts-ignore
            if (navigator.sendBeacon) navigator.sendBeacon(`${constants.API_GAMES}/${gameId}/leave`, blob);
          } catch {}
        });
      } catch {}
    };

    const onBeforeUnload = () => leaveIfEligible();
    const onPageHide = () => leaveIfEligible();

    window.addEventListener('beforeunload', onBeforeUnload);
    window.addEventListener('pagehide', onPageHide);

    return () => {
      window.removeEventListener('beforeunload', onBeforeUnload);
      window.removeEventListener('pagehide', onPageHide);
      leaveIfEligible();
    };
  }, [gameId, currentPlayerUUID]);

  const handleStartGame = async () => {
    try {
      if (actingRef.current || submitting) return;
      actingRef.current = true;
      setSubmitting(true);
      const response = await apiFetch(`${constants.API_GAMES}/${gameId}/start`, { method: 'POST' });
      if (!response.ok) {
        throw new Error('Failed to start game');
      }
      // Request accepted; keep the local submitting state true so the
      // Start button remains disabled until polling detects the game has
      // transitioned to `in_progress` (or an error occurs).
      actingRef.current = false;
      return;
    } catch (err: any) {
      alert(`Error starting game: ${err.message}`);
      console.error(err);
      // Re-enable only on error so user can retry
      setSubmitting(false);
      actingRef.current = false;
    }
  };

  if (error) {
    return <div>Error: {error}</div>;
  }

  if (!game) {
    return <div>Loading game...</div>;
  }

  const isCreator = game.players.length > 0 && game.players[0].player_uuid === currentPlayerUUID;
  const currentPlayer: Player | undefined = game.players.find(p => p.player_uuid === currentPlayerUUID);
  const allReady = game.players.length === 2 && game.players.every(p => p.has_created);

  const leaveGameAndReturn = async () => {
    try {
      if (game?.status === 'waiting_for_players' && currentPlayerUUID && game.players?.some(p => p.player_uuid === currentPlayerUUID)) {
                await apiFetch(`${constants.API_GAMES}/${gameId}/leave`, {
          method: 'POST',
          headers: { [constants.HEADER_CONTENT_TYPE]: constants.CONTENT_TYPE_JSON },
          body: JSON.stringify({ player_uuid: currentPlayerUUID }),
        });
        
      }
    } catch (e) {
      console.warn('Leave failed (continuing to lobby):', e);
    } finally {
      try { localStorage.removeItem('game_id'); } catch {}
      navigate('/');
    }
  };

  return (
    <div>
      <header className="page-header">
        <h3>Game Room #{game.ID} (Code: {game.join_code})</h3>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <button onClick={leaveGameAndReturn}>Back to Lobby</button>
          <SettingsMenu />
        </div>
      </header>

      <main style={{ padding: '20px' }}>
        <h4>Status: {game.status}</h4>

        <h4>Players ({game.players?.length || 0} / 2)</h4>
        <ul style={{ listStyle: 'none', padding: 0 }}>
          {game.players.map(player => (
            <li key={player.ID} style={{ border: '1px solid #444', padding: '10px', marginBottom: '10px', borderRadius: '5px' }}>
              {player.player_name || `Player ${player.ID}`} ({player.player_uuid}) {player.player_uuid === currentPlayerUUID ? '(You)' : ''}
              <div style={{ fontSize: 12, color: '#ccc' }}>Hybrids created: {player.has_created ? 'Yes' : 'No'}</div>
            </li>
          ))}
        </ul>

        {game.status === 'waiting_for_players' && currentPlayer && !currentPlayer.has_created && (
          <HybridCreation gameId={gameId!} onCreated={() => {}} />
        )}

        {isCreator && game.players.length === 2 && game.status === 'waiting_for_players' && (
          <button onClick={handleStartGame} disabled={!allReady || submitting} style={{ padding: '10px 20px', fontSize: '16px' }}>
            {allReady ? 'Start Game' : 'Waiting hybrids...'}
          </button>
        )}

        {game.status === 'waiting_for_players' && !isCreator && (
          <p>Waiting for the host to start the game...</p>
        )}

        {(game.status === 'starting' || submitting) && (
          <p>Your hybrid is being created. This may take a few moments.</p>
        )}

        {game.status === 'in_progress' && (
          <p>Game is in progress. Redirecting to game board...</p>
        )}

        <div style={{ marginTop: 12 }}>
          <button
            onClick={async () => {
              if (submitting) return;
              setSubmitting(true);
              try {
                // Before the game starts, just leave to free up the slot
                if (game?.status === 'waiting_for_players') {
                                    await apiFetch(`${constants.API_GAMES}/${gameId}/leave`, {
                    method: 'POST',
                    headers: { [constants.HEADER_CONTENT_TYPE]: constants.CONTENT_TYPE_JSON },
                    body: JSON.stringify({ player_uuid: currentPlayerUUID }),
                  });
                  
                } else {
                  // Fallback: end the match if somehow already started from this view
                                    await apiFetch(`${constants.API_GAMES}/${gameId}/end`, { method: 'POST' });
                }
              } catch (e) {
                console.warn('Cancel failed, continuing to lobby:', e);
              } finally {
                setSubmitting(false);
                try { localStorage.removeItem('game_id'); } catch {}
                navigate('/');
              }
            }}
            disabled={submitting}
          >
            Cancel Match
          </button>
        </div>
      </main>
    </div>
  );
};

export default GameRoom;
