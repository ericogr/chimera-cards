import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Entity } from './types';
import { apiFetch } from './api';
import * as constants from './constants';
import { entityAssetUrl } from './utils/keys';
import './HybridCreation.css';
import { Button, Modal } from './ui';

interface Props {
  gameCode: string;
  onCreated?: () => void;
  // When true, the server-side public-games TTL expired and creation must be disabled
  ttlExpired?: boolean;
}

interface HybridSpecState {
  entityIds: number[];
  selectedEntityId?: number;
}

const HybridCreation: React.FC<Props> = ({ gameCode, onCreated, ttlExpired = false }) => {
  const [entities, setEntities] = useState<Entity[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [h1, setH1] = useState<HybridSpecState>({ entityIds: [], selectedEntityId: undefined });
  const [h2, setH2] = useState<HybridSpecState>({ entityIds: [], selectedEntityId: undefined });
  const [submitting, setSubmitting] = useState(false);
  const actingRef = useRef(false);
  const playerUUID = localStorage.getItem('player_uuid') || '';
  const [showHelp, setShowHelp] = useState(false);

  useEffect(() => {
    const fetchEntities = async () => {
      try {
        const res = await apiFetch(constants.API_ENTITIES);
        if (!res.ok) throw new Error('Failed to load entities');
        const data: Entity[] = await res.json();
        setEntities(data);
      } catch (e: any) {
        setError(e.message || 'Error loading entities');
      } finally {
        setLoading(false);
      }
    };
    fetchEntities();
  }, []);

  const usedIds = useMemo(() => new Set([...h1.entityIds, ...h2.entityIds]), [h1, h2]);
  const toggleAnimalSelection = (target: 'h1' | 'h2', id: number) => {
    if (submitting || ttlExpired) return;
    const src = target === 'h1' ? h1 : h2;
    const setter = target === 'h1' ? setH1 : setH2;
    const isUsedElsewhere = usedIds.has(id) && !src.entityIds.includes(id);
    if (isUsedElsewhere) return;
    const picked = src.entityIds.includes(id)
      ? src.entityIds.filter((x) => x !== id)
      : src.entityIds.length < 3
      ? [...src.entityIds, id]
      : src.entityIds; // allow up to 3
    const updated = { ...src, entityIds: picked } as HybridSpecState;
    // Reset selected ability if it no longer belongs to the chosen set
    if (updated.selectedEntityId && !updated.entityIds.includes(updated.selectedEntityId)) {
      updated.selectedEntityId = undefined;
    }
    setter(updated);
  };

  const isValidSelection =
    h1.entityIds.length >= 2 && h1.entityIds.length <= 3 &&
    h2.entityIds.length >= 2 && h2.entityIds.length <= 3 &&
    h1.entityIds.every((id) => !h2.entityIds.includes(id)) &&
    !!h1.selectedEntityId &&
    !!h2.selectedEntityId;

  

  const handleSubmit = async () => {
    if (ttlExpired) return;
    if (!isValidSelection || actingRef.current || submitting) return;
    actingRef.current = true;
    setSubmitting(true);
    try {
      const res = await apiFetch(`${constants.API_GAMES}/${gameCode}/create-hybrids`, {
        method: 'POST',
        headers: { [constants.HEADER_CONTENT_TYPE]: constants.CONTENT_TYPE_JSON },
      body: JSON.stringify({
          player_uuid: playerUUID,
          hybrid1: { entity_ids: h1.entityIds, selected_entity_id: h1.selectedEntityId },
          hybrid2: { entity_ids: h2.entityIds, selected_entity_id: h2.selectedEntityId },
        }),
      });
      if (!res.ok) {
        const msg = await res.text();
        throw new Error(msg || 'Failed to create hybrids');
      }
      onCreated?.();
      return;
    } catch (e: any) {
      alert(`Error: ${e.message}`);
      setSubmitting(false);
      actingRef.current = false;
    }
  };

  if (loading) return <div>Loading entities...</div>;
  if (error) return <div>Error: {error}</div>;

  const imageSrcFor = (name: string) => {
    return entityAssetUrl(name);
  };

  const animalCard = (a: Entity, target: 'h1' | 'h2') => {
    const src = target === 'h1' ? h1 : h2;
    const disabled = usedIds.has(a.ID) && !src.entityIds.includes(a.ID);
    const selected = src.entityIds.includes(a.ID);
    const canSelect = !disabled && (selected || src.entityIds.length < 3);
    const needsAbilitySelection = src.entityIds.length > 0 && src.selectedEntityId === undefined && src.entityIds.includes(a.ID);
    return (
      <div
        key={`${target}-${a.ID}`}
        onClick={() => toggleAnimalSelection(target, a.ID)}
        className={`hybrid-entity-card ${disabled ? 'disabled' : ''} ${selected ? 'selected' : ''} ${canSelect ? 'blink-border' : ''}`}
      >
        {/* Top content: image + info */}
        <div className="row-center-sm">
          <img
            src={imageSrcFor(a.name)}
            alt={a.name}
            width={96}
            height={96}
            className={`entity-image`}
            onError={(e) => { (e.currentTarget as HTMLImageElement).style.visibility = 'hidden'; }}
          />
          <div className="flex-1">
            <strong className="block">{a.name}</strong>
            <div className="muted-sm">
              HP {a.pv} | ATK {a.atq} | DEF {a.def} | AGI {a.agi} | ENE {a.ene} | VIG {a.vigor_cost ?? '-'}
            </div>
            <div className="muted-sm">{a.skill?.name} (Cost {a.skill?.cost})</div>
            {(() => {
              const isPicked = src.entityIds.includes(a.ID);
              return (
                <div className={`ability-row`} onClick={isPicked ? (e) => e.stopPropagation() : undefined}>
                  {isPicked && (
                    <label className="label-inline">
                      <input
                        type="radio"
                        name={`${target}-selected-ability`}
                        checked={src.selectedEntityId === a.ID}
                        onChange={() => {
                          const setter = target === 'h1' ? setH1 : setH2;
                          setter({ ...src, selectedEntityId: a.ID });
                        }}
                        disabled={!selected}
                      />
                      <span className={needsAbilitySelection ? 'blink-text' : ''}>Use this entity's special ability</span>
                    </label>
                  )}
                </div>
              );
            })()}
          </div>
        </div>
        {/* Radio moved under description to minimize vertical space */}
      </div>
    );
  };

  const grid = (target: 'h1' | 'h2') => (
    <div className="entities-grid">
      {entities.map((a) => animalCard(a, target))}
    </div>
  );

  return (
    <div className="card hybrid-creation-card">
      <div className="row-center">
        <Button type="button" variant="ghost" onClick={() => setShowHelp(true)}>
          Help
        </Button>
        <h3 className="no-margin">Create Your Hybrids</h3>
      </div>
      <div className="hybrid-creation-grid">
        <section>
          <h4>Hybrid 1</h4>
          {grid('h1')}
          <div className="muted-sm mt-4">Pick 2 to 3 entities and choose 1 special ability among them</div>
          
        </section>
        <section>
          <h4>Hybrid 2</h4>
          {grid('h2')}
          <div className="muted-sm mt-4">Pick 2 to 3 entities (no overlap with Hybrid 1) and choose 1 special ability</div>
          
        </section>
        <Button onClick={handleSubmit} disabled={!isValidSelection || submitting || ttlExpired}>
          {submitting ? 'Creating…' : 'Create Hybrids'}
        </Button>
        {ttlExpired && (
          <div className="error-sm">Hybrid creation is closed — the game can no longer be started.</div>
        )}
        
      </div>
      {showHelp && (
        <Modal onClose={() => setShowHelp(false)}>
          <div className="text-left">
            <ul className="help-list">
              <li className="mb-8">Pick 2–3 entities for each hybrid.</li>
              <li className="mb-8">Create two different hybrids — hybrids cannot share the same entity.</li>
              <li className="mb-8">For each hybrid, choose one of the selected entities to enable its special ability.</li>
              <li>Tap <strong>"Create Hybrids"</strong> to save.</li>
            </ul>
          </div>
          <div className="mt-8 muted">Tap anywhere to close</div>
        </Modal>
      )}
    </div>
  );
};

export default HybridCreation;
