import React, { useEffect, useRef, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import HybridCreation from './HybridCreation';
import Timer from './Timer';
import { useGame } from './hooks/useGame';
import { Button, WaitingAnimation } from './ui';
import { Player } from './types';
import { apiFetch } from './api';
import * as constants from './constants';
import { safeRemoveLocal } from './runtimeConfig';

const GameRoom: React.FC = () => {
  const { gameId } = useParams<{ gameId: string }>();
  const navigate = useNavigate();
  const { game, error: gameError } = useGame(gameId, 3000);
  const [timeLeftMs, setTimeLeftMs] = useState<number | null>(null);
  const [publicGamesTTLSeconds, setPublicGamesTTLSeconds] = useState<number | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const actingRef = useRef(false);
  const hasLeftRef = useRef(false);
  const toBoardRef = useRef(false);

  const currentPlayerUUID = localStorage.getItem('player_uuid');

  // react to game updates from the polling hook
  useEffect(() => {
    if (!game) return;
    if (game.status === 'starting') {
      setSubmitting(true);
    } else if (game.status === 'in_progress') {
      setSubmitting(false);
      toBoardRef.current = true;
      navigate(`/game/${gameId}/board`);
    } else if (game.status === 'error') {
      setSubmitting(false);
    } else {
      setSubmitting(false);
    }
  }, [game, gameId, navigate]);

  // Fetch backend config (public games TTL) and compute countdown.
  useEffect(() => {
    let mounted = true;
    const loadConfig = async () => {
      try {
        const res = await apiFetch(constants.API_CONFIG);
        if (!res.ok) return;
        const body = await res.json();
        if (!mounted) return;
        if (body && typeof body.public_games_ttl_seconds === 'number') {
          setPublicGamesTTLSeconds(body.public_games_ttl_seconds);
        }
      } catch (e) {
        // ignore
      }
    };
    loadConfig();
    return () => {
      mounted = false;
    };
  }, []);

  useEffect(() => {
    if (!game) {
      setTimeLeftMs(null);
      return;
    }
    if (game.private) {
      setTimeLeftMs(null);
      return;
    }

    const ttlSec = publicGamesTTLSeconds ?? 300; // default 5m
    const ttlMs = ttlSec * 1000;
    const createdAt = game.created_at ? new Date(game.created_at).getTime() : NaN;
    if (isNaN(createdAt)) {
      setTimeLeftMs(null);
      return;
    }
    const expiry = createdAt + ttlMs;

    const update = () => setTimeLeftMs(Math.max(0, expiry - Date.now()));
    update();
    const id = setInterval(update, 1000);
    return () => clearInterval(id);
  }, [game, publicGamesTTLSeconds]);

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

  const effectiveError = gameError;
  if (effectiveError) {
    return <div>Error: {effectiveError}</div>;
  }

  if (!game) {
    return <div>Loading game...</div>;
  }

  const isCreator = game.players.length > 0 && game.players[0].player_uuid === currentPlayerUUID;
  const currentPlayer: Player | undefined = game.players.find(p => p.player_uuid === currentPlayerUUID);
  const allReady = game.players.length === 2 && game.players.every(p => p.has_created);

  

  const leaveGameAndReturn = async () => {
    try {
      // Prevent the unload/visibility handler from firing an extra leave
      hasLeftRef.current = true;
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
      safeRemoveLocal('game_id');
      navigate('/');
    }
  };

  return (
    <div>
      <main className="page-main">
        <div className="row-between">
          <h3 className="no-margin">Game Room #{game.ID} (Code: {game.join_code})</h3>
          <div>
            <Button onClick={leaveGameAndReturn}>Back to Lobby</Button>
          </div>
        </div>

        {/* Countdown until public game TTL expires (only for public games in waiting state) */}
        {game && !game.private && timeLeftMs !== null && game.status === 'waiting_for_players' && (
          <div className="muted small mt-6">
            {timeLeftMs > 0 ? (
              <>
                Time left to start: <Timer seconds={Math.floor((timeLeftMs || 0) / 1000)} />
              </>
            ) : (
              'This public game can no longer be started.'
            )}
          </div>
        )}

        <h4>Status: {game.status}</h4>

        <h4>Players ({game.players?.length || 0} / 2)</h4>
        <ul className="list-reset">
          {game.players.map(player => (
            <li key={player.ID} className="player-card">
              {player.player_name || `Player ${player.ID}`} ({player.player_uuid}) {player.player_uuid === currentPlayerUUID ? '(You)' : ''}
              <div className="muted-sm">Hybrids created: {player.has_created ? 'Yes' : 'No'}</div>
            </li>
          ))}
        </ul>

        {game.status === 'waiting_for_players' && currentPlayer && !currentPlayer.has_created && (
          <HybridCreation gameId={gameId!} onCreated={() => {}} ttlExpired={timeLeftMs !== null && timeLeftMs <= 0} />
        )}

        {isCreator && game.players.length === 2 && game.status === 'waiting_for_players' && (
          <Button onClick={handleStartGame} disabled={!allReady || submitting || (timeLeftMs !== null && timeLeftMs <= 0)}>
            {allReady ? 'Start Game' : 'Waiting hybrids...'}
          </Button>
        )}

        {game.status === 'waiting_for_players' && !isCreator && (
          <p>Waiting for the host to start the game...</p>
        )}

        {(game.status === 'starting' || submitting) && (
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center' }} className="mt-6">
            <WaitingAnimation size={192} />
            <div className="muted-sm mt-4">Your hybrid is being created and this may take up to 2 minutes.</div>
          </div>
        )}

        {game.status === 'in_progress' && (
          <p>Game is in progress. Redirecting to game board...</p>
        )}

        <div className="mt-12">
          <Button
            onClick={async () => {
              if (submitting) return;
              // Prevent the unload/visibility handler from firing an extra leave
              hasLeftRef.current = true;
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
                safeRemoveLocal('game_id');
                navigate('/');
              }
            }}
            disabled={submitting}
          >
            Cancel Match
          </Button>
        </div>
      </main>
    </div>
  );
};

export default GameRoom;
