import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Entity } from './types';
import { apiFetch } from './api';
import * as constants from './constants';
import { entityAssetUrl } from './utils/keys';

interface Props {
  gameId: string;
  onCreated?: () => void;
}

interface HybridSpecState {
  entityIds: number[];
  selectedEntityId?: number;
}

const HybridCreation: React.FC<Props> = ({ gameId, onCreated }) => {
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
    if (submitting) return;
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

  const idToName = new Map(entities.map(a => [a.ID, a.name] as const));
  const computeName = (ids: number[]) => {
    if (ids.length === 0) return '';
    const names = ids.map(id => idToName.get(id) || '').filter(Boolean).sort((a, b) => a.localeCompare(b, 'pt-BR'));
    return names.join(' + ');
  };
  const h1Name = computeName(h1.entityIds);
  const h2Name = computeName(h2.entityIds);

  const handleSubmit = async () => {
    if (!isValidSelection || actingRef.current || submitting) return;
    actingRef.current = true;
    setSubmitting(true);
    try {
      const res = await apiFetch(`${constants.API_GAMES}/${gameId}/create-hybrids`, {
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
    return (
      <div
        key={`${target}-${a.ID}`}
        onClick={() => toggleAnimalSelection(target, a.ID)}
        style={{
          border: '1px solid #444',
          padding: '8px',
          borderRadius: 6,
          opacity: disabled ? 0.4 : 1,
          background: selected ? '#1f3b' : 'transparent',
          cursor: disabled ? 'not-allowed' : 'pointer',
          display: 'flex',
          flexDirection: 'column',
          gap: 8,
        }}
      >
        {/* Top content: image + info */}
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <img
            src={imageSrcFor(a.name)}
            alt={a.name}
            width={96}
            height={96}
            style={{ objectFit: 'cover', borderRadius: 6, border: selected ? '2px solid #61dafb' : '2px solid transparent' }}
            onError={(e) => { (e.currentTarget as HTMLImageElement).style.visibility = 'hidden'; }}
          />
          <div style={{ flex: 1, minWidth: 0 }}>
            <strong style={{ display: 'block' }}>{a.name}</strong>
            <div style={{ fontSize: 12, color: '#ccc' }}>
              HP {a.pv} | ATK {a.atq} | DEF {a.def} | AGI {a.agi} | ENE {a.ene} | VIG {a.vigor_cost ?? '-'}
            </div>
            <div style={{ fontSize: 12 }}>{a.skill?.name} (Cost {a.skill?.cost})</div>
            {(() => {
              const isPicked = src.entityIds.includes(a.ID);
              return (
                <div
                  style={{
                    marginTop: 4,
                    borderTop: `1px dashed ${isPicked ? '#ccc' : 'transparent'}`,
                    paddingTop: 4,
                    minHeight: 22,
                  }}
                  onClick={isPicked ? (e) => e.stopPropagation() : undefined}
                >
                  {isPicked && (
                    <label style={{ fontSize: 12, display: 'inline-flex', alignItems: 'center', gap: 6 }}>
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
                      <span>Use this entity's special ability</span>
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
    <div style={{ border: '1px solid #333', padding: 16, borderRadius: 8, position: 'relative' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <button
          type="button"
          onClick={() => setShowHelp(true)}
          style={{
            background: 'transparent',
            color: '#61dafb',
            border: '1px solid #61dafb',
            padding: '6px 10px',
            borderRadius: 6,
            cursor: 'pointer',
            fontSize: 14,
          }}
        >
          Help
        </button>
        <h3 style={{ margin: 0 }}>Create Your Hybrids</h3>
      </div>
      <div className="hybrid-creation-grid">
        <section>
          <h4>Hybrid 1</h4>
          {grid('h1')}
          <div style={{ fontSize: 12, marginTop: 4 }}>Pick 2 to 3 entities and choose 1 special ability among them</div>
          <div style={{ fontSize: 12, marginTop: 4, color: '#ccc' }}>Name (auto): {h1Name || '-'}</div>
        </section>
        <section>
          <h4>Hybrid 2</h4>
          {grid('h2')}
          <div style={{ fontSize: 12, marginTop: 4 }}>Pick 2 to 3 entities (no overlap with Hybrid 1) and choose 1 special ability</div>
          <div style={{ fontSize: 12, marginTop: 4, color: '#ccc' }}>Name (auto): {h2Name || '-'}</div>
        </section>
        <button onClick={handleSubmit} disabled={!isValidSelection || submitting} style={{ padding: '10px 16px' }}>
          {submitting ? 'Creating…' : 'Create Hybrids'}
        </button>
        <div style={{ fontSize: 12, color: '#aaa' }}>* Names are generated automatically by the game.</div>
      </div>
      {showHelp && (
        <div
          onClick={() => setShowHelp(false)}
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            background: 'rgba(0,0,0,0.7)',
            zIndex: 2000,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            padding: 20,
          }}
        >
          <div
            style={{
              background: '#fff',
              color: '#000',
              borderRadius: 8,
              padding: '18px 20px',
              maxWidth: 680,
              width: '100%',
              boxSizing: 'border-box',
              textAlign: 'center',
              fontSize: 16,
              lineHeight: 1.4,
            }}
          >
            <div style={{ textAlign: 'left' }}>
              <ul style={{ paddingLeft: 18, margin: 0 }}>
                <li style={{ marginBottom: 8 }}>Pick 2–3 entities for each hybrid.</li>
                <li style={{ marginBottom: 8 }}>Create two different hybrids — hybrids cannot share the same entity.</li>
                <li style={{ marginBottom: 8 }}>For each hybrid, choose one of the selected entities to enable its special ability.</li>
                <li>Tap <strong>"Create Hybrids"</strong> to save.</li>
              </ul>
            </div>
            <div style={{ marginTop: 8, fontSize: 13, color: '#666' }}>Tap anywhere to close</div>
          </div>
        </div>
      )}
    </div>
  );
};

export default HybridCreation;
