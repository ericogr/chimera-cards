import React, { useState, useEffect, useRef } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import './GameBoard.css';
import iconAttack from './images/basic_attack.svg';
import iconDefend from './images/defend.svg';
import iconRest from './images/rest.svg';
import iconAbility from './images/ability.svg';
import iconEnd from './images/end_match.svg';
import { Game, Player, Hybrid, Entity, EntityName } from './types';
import { hybridAssetUrlFromNames } from './utils/keys';
import { apiFetch } from './api';
import * as constants from './constants';

const GameBoard: React.FC = () => {
  const { gameId } = useParams<{ gameId: string }>();
  const navigate = useNavigate();
  const [game, setGame] = useState<Game | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [lockedRound, setLockedRound] = useState<number | null>(null);
  const actingRef = useRef(false);
  const endRef = useRef(false);
  const playerUUID = localStorage.getItem('player_uuid') || '';
  const playerEmail = ((): string => {
    try { return localStorage.getItem('player_email') || ''; } catch { return ''; }
  })();

  useEffect(() => {
    const fetchGame = async () => {
      try {
        const response = await apiFetch(`${constants.API_GAMES}/${gameId}`);
        if (!response.ok) {
          throw new Error('Game not found or an error occurred');
        }
        const data = await response.json();
        setGame(data);
      } catch (err) {
        setError('Could not load game data.');
        console.error(err);
      }
    };

    fetchGame(); // Initial fetch

    const interval = setInterval(fetchGame, 3000); // Poll every 3 seconds

    return () => clearInterval(interval); // Cleanup on unmount
  }, [gameId]);

  useEffect(() => {
    if (!game) return;
    if (lockedRound !== null && game.round_count > lockedRound) {
      setLockedRound(null);
      setSubmitting(false);
      actingRef.current = false;
    }
  }, [game, lockedRound]);

  if (error) {
    return <div className="game-board-error">Error: {error}</div>;
  }

  if (!game) {
    return <div className="game-board-loading">Loading game...</div>;
  }

  const [player1, player2] = game.players;
  const me: Player | undefined = game.players.find(p => p.player_uuid === playerUUID);
  const opponent: Player | undefined = game.players.find(p => p.player_uuid !== playerUUID);
  const myActive: Hybrid | undefined = me?.hybrids?.find(h => h.is_active && !h.is_defeated);
  const planning = game.status === 'in_progress' && game.phase === 'planning';
  const myTurn = planning && !me?.has_submitted_action;

  const submittedLabel = (p?: Player) => (p?.has_submitted_action ? 'Submitted' : 'Waiting');

  const vigCostFor = (animalName: string) => {
    switch (animalName) {
      case EntityName.Lion:
      case EntityName.Cheetah:
      case EntityName.Octopus:
        return 2;
      case EntityName.Bear:
      case EntityName.Rhino:
      case EntityName.Turtle:
      case EntityName.Gorilla:
        return 3;
      case EntityName.Eagle:
      case EntityName.Wolf:
      case EntityName.Raven:
        return 1;
      default:
        return 2;
    }
  };

  const currentActionLabel = () => {
    if (!planning || !me?.pending_action_type) return '';
    if (me.pending_action_type === 'ability') {
      const entity = myActive?.base_entities?.find(a => a.ID === me.pending_action_entity_id);
      return entity ? `Ability: ${entity.skill?.name || 'Ability'}` : 'Ability';
    }
    if (me.pending_action_type === 'basic_attack') return 'Basic Attack';
    if (me.pending_action_type === 'defend') return 'Defend';
    if (me.pending_action_type === 'rest') return 'Rest';
    return '';
  };

  const submitAction = async (action_type: 'basic_attack' | 'defend' | 'ability' | 'rest', entity?: Entity) => {
    try {
      if (actingRef.current || submitting || me?.has_submitted_action) return;
      actingRef.current = true;
      setSubmitting(true);
      if (game?.round_count != null) {
        setLockedRound(game.round_count);
      }
      const res = await apiFetch(`${constants.API_GAMES}/${gameId}/action`, {
        method: 'POST',
        headers: { [constants.HEADER_CONTENT_TYPE]: constants.CONTENT_TYPE_JSON },
        body: JSON.stringify({ player_uuid: playerUUID, action_type, entity_id: entity?.ID }),
      });
      if (!res.ok) throw new Error(await res.text());
    } catch (e: any) {
      alert(`Action error: ${e.message}`);
    } finally { /* keep locked until next round */ }
  };

  return (
    <div className="game-board-container">
      <header className="game-board-header" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12 }}>
        <div className="game-info">
          <h1>{game.name}</h1>
          <p>{game.description}</p>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 14 }}>
          <div className="game-meta" style={{ textAlign: 'right' }}>
            <p>Game ID: {game.ID}</p>
            <p>Created: {new Date(game.created_at).toLocaleString()}</p>
          </div>
        </div>
      </header>

      <main className="game-board-main">
        <div className="player-area player-one">
          <h2>{player1?.player_name || 'Waiting for Player 1'}</h2>
          {player1 && (
            <div>
              {(() => {
                const active = player1.hybrids?.find(h => h.is_active);
                return (
                  <div>
                    <p>Active: {active?.generated_name || active?.name || '-'}</p>
                    <p style={{ fontSize: 12, color: '#bbb' }}>Combination: {active?.name || '-'}</p>
                  </div>
                );
              })()}
              <Stats hybrid={player1.hybrids?.find(h => h.is_active)} isMe={player1.player_uuid===playerUUID} />
            </div>
          )}
        </div>
        <div className="player-area player-two">
          <h2>{player2?.player_name || 'Waiting for Player 2'}</h2>
          {player2 && (
            <div>
              {(() => {
                const active = player2.hybrids?.find(h => h.is_active);
                return (
                  <div>
                    <p>Active: {active?.generated_name || active?.name || '-'}</p>
                    <p style={{ fontSize: 12, color: '#bbb' }}>Combination: {active?.name || '-'}</p>
                  </div>
                );
              })()}
              <Stats hybrid={player2.hybrids?.find(h => h.is_active)} isMe={player2.player_uuid===playerUUID} />
            </div>
          )}
        </div>
      </main>

      <footer className="game-board-footer">
        <div>Round: {game.round_count} | Phase: {game.phase || '-'} | {myTurn ? 'Choose your action' : planning ? 'Waiting opponent/you' : 'Resolving...'}</div>
        <div style={{ marginTop: 6, fontSize: 13, color: '#ccc' }}>
          Your action: {submittedLabel(me)} | Opponent action: {submittedLabel(opponent)}
        </div>
        {/* End Match moved to the bottom as a final action with consistent layout */}
        {game.status === 'finished' && (
          <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
            <div>Winner: {game.winner || me?.player_name || '-'}</div>
            <button
              onClick={() => {
                try { localStorage.removeItem('game_id'); } catch {}
                navigate('/');
              }}
            >
              Back to Lobby
            </button>
          </div>
        )}
        {myTurn && myActive && (
          <div className="action-panel" style={{ marginTop: 12 }}>
            <div className="action-row">
              <button className="icon-btn" onClick={() => submitAction('basic_attack')} disabled={submitting || !!me?.has_submitted_action || lockedRound !== null}>
                <img src={iconAttack} alt="Basic Attack" className="btn-icon" />
                Basic Attack
              </button>
              <div className="action-desc">Perform a basic attack with your active hybrid.</div>
            </div>
            <div className="action-row">
              <button className="icon-btn" onClick={() => submitAction('defend')} disabled={submitting || !!me?.has_submitted_action || lockedRound !== null}>
                <img src={iconDefend} alt="Defend" className="btn-icon" />
                Defend
              </button>
              <div className="action-desc">Increase defense this round. Spends VIG if available.</div>
            </div>
            {(() => {
              const selId = myActive.selected_ability_entity_id;
              const ability = myActive.base_entities?.find(a => a.ID === selId) as Entity | undefined;
              if (!ability) return null;
              const notEnoughEnergy = (myActive?.current_ene || 0) < (ability.skill?.cost || 0);
              return (
                <div className="action-row">
                  <button className="icon-btn" onClick={() => submitAction('ability', ability)} disabled={submitting || !!me?.has_submitted_action || lockedRound !== null || notEnoughEnergy}>
                    <img src={iconAbility} alt="Ability" className="btn-icon" />
                    {ability.skill?.name}
                  </button>
                  <div className="action-desc">
                    {ability.skill?.description} — ENE {ability.skill?.cost}, VIG {vigCostFor(ability.name)}
                  </div>
                </div>
              );
            })()}
            <div className="action-row">
              <button className="icon-btn" onClick={() => submitAction('rest')} disabled={submitting || !!me?.has_submitted_action || lockedRound !== null}>
                <img src={iconRest} alt="Rest" className="btn-icon" />
                Rest
              </button>
              <div className="action-desc">Recover +2 VIG and +2 ENE.</div>
            </div>
          </div>
        )}
        {!myTurn && planning && (
          <div style={{ marginTop: 8 }}>{me?.has_submitted_action ? 'You already chose. Waiting for opponent...' : 'Waiting for both actions...'}</div>
        )}
        

        <div className="action-row" style={{ marginTop: 12 }}>
          <button
            onClick={async () => {
              try {
                if (endRef.current || submitting) return;
                endRef.current = true;
                setSubmitting(true);
                await apiFetch(`${constants.API_GAMES}/${gameId}/end`, {
                  method: 'POST',
                  headers: { [constants.HEADER_CONTENT_TYPE]: constants.CONTENT_TYPE_JSON },
                  body: JSON.stringify({ player_uuid: playerUUID, player_email: playerEmail }),
                });
              } finally {
                setSubmitting(false);
                endRef.current = false;
                try { localStorage.removeItem('game_id'); } catch {}
                navigate('/');
              }
            }}
            disabled={submitting}
            className="icon-btn"
          >
            <img src={iconEnd} alt="End" className="btn-icon" />
            End Match
          </button>
          <div className="action-desc">Forfeit the match — ends combat and records a resignation for your player (no victory awarded to the opponent).</div>
        </div>

        {planning && (
          <div style={{ marginTop: 8, fontSize: 13, color: '#ccc' }}>
            Your choice: {currentActionLabel() || '—'}
          </div>
        )}

        {game.last_round_summary && (
          <div style={{ marginTop: 12, whiteSpace: 'pre-wrap', padding: 12, background: '#111', border: '1px solid #333', borderRadius: 6 }}>
            <strong>Last Round:</strong>
            <div>{game.last_round_summary}</div>
          </div>
        )}
      </footer>
    </div>
  );
};

export default GameBoard;

const Stats: React.FC<{ hybrid?: Hybrid; isMe: boolean }> = ({ hybrid, isMe }) => {
  if (!hybrid) return <div />;

  const imgSrc = hybridAssetUrlFromNames((hybrid?.base_entities || []).map(a => a.name));

  return (
    <div style={{ display: 'flex', gap: 12, alignItems: 'flex-start' }}>
      {imgSrc && (
        <img src={imgSrc} alt={hybrid.generated_name || hybrid.name} width={96} height={96} style={{ objectFit: 'cover', borderRadius: 6 }} onError={(e)=>{ (e.currentTarget as HTMLImageElement).style.visibility = 'hidden'; }} />
      )}
      <div className="stats-grid">
        <div>HP: {hybrid.current_pv} / {hybrid.base_pv}</div>
        <div>ATK: {hybrid.current_atq}</div>
        <div>DEF: {hybrid.current_def}</div>
        <div>AGI: {hybrid.current_agi}</div>
        <div>ENE: {hybrid.current_ene}</div>
        {'current_vig' in hybrid && <div>VIG: {hybrid.current_vig} {hybrid.base_vig ? `/ ${hybrid.base_vig}` : ''}</div>}
      </div>
    </div>
  );
};
