import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Animal } from './types';
import { apiFetch } from './api';
import * as constants from './constants';
import { animalAssetUrl } from './utils/keys';

interface Props {
  gameId: string;
  onCreated?: () => void;
}

interface HybridSpecState {
  animalIds: number[];
  selectedAnimalId?: number;
}

const HybridCreation: React.FC<Props> = ({ gameId, onCreated }) => {
  const [animals, setAnimals] = useState<Animal[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [h1, setH1] = useState<HybridSpecState>({ animalIds: [], selectedAnimalId: undefined });
  const [h2, setH2] = useState<HybridSpecState>({ animalIds: [], selectedAnimalId: undefined });
  const [submitting, setSubmitting] = useState(false);
  const actingRef = useRef(false);
  const playerUUID = localStorage.getItem('player_uuid') || '';

  useEffect(() => {
    const fetchAnimals = async () => {
      try {
        const res = await apiFetch(constants.API_ANIMALS);
        if (!res.ok) throw new Error('Failed to load animals');
        const data: Animal[] = await res.json();
        setAnimals(data);
      } catch (e: any) {
        setError(e.message || 'Error loading animals');
      } finally {
        setLoading(false);
      }
    };
    fetchAnimals();
  }, []);

  const usedIds = useMemo(() => new Set([...h1.animalIds, ...h2.animalIds]), [h1, h2]);
  const toggleAnimalSelection = (target: 'h1' | 'h2', id: number) => {
    if (submitting) return;
    const src = target === 'h1' ? h1 : h2;
    const setter = target === 'h1' ? setH1 : setH2;
    const isUsedElsewhere = usedIds.has(id) && !src.animalIds.includes(id);
    if (isUsedElsewhere) return;
    const picked = src.animalIds.includes(id)
      ? src.animalIds.filter((x) => x !== id)
      : src.animalIds.length < 3
      ? [...src.animalIds, id]
      : src.animalIds; // allow up to 3
    const updated = { ...src, animalIds: picked } as HybridSpecState;
    // Reset selected ability if it no longer belongs to the chosen set
    if (updated.selectedAnimalId && !updated.animalIds.includes(updated.selectedAnimalId)) {
      updated.selectedAnimalId = undefined;
    }
    setter(updated);
  };

  const isValidSelection =
    h1.animalIds.length >= 2 && h1.animalIds.length <= 3 &&
    h2.animalIds.length >= 2 && h2.animalIds.length <= 3 &&
    h1.animalIds.every((id) => !h2.animalIds.includes(id)) &&
    !!h1.selectedAnimalId &&
    !!h2.selectedAnimalId;

  const idToName = new Map(animals.map(a => [a.ID, a.name] as const));
  const computeName = (ids: number[]) => {
    if (ids.length === 0) return '';
    const names = ids.map(id => idToName.get(id) || '').filter(Boolean).sort((a, b) => a.localeCompare(b, 'pt-BR'));
    return names.join(' + ');
  };
  const h1Name = computeName(h1.animalIds);
  const h2Name = computeName(h2.animalIds);

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
          hybrid1: { animal_ids: h1.animalIds, selected_animal_id: h1.selectedAnimalId },
          hybrid2: { animal_ids: h2.animalIds, selected_animal_id: h2.selectedAnimalId },
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

  if (loading) return <div>Loading animals...</div>;
  if (error) return <div>Error: {error}</div>;

  const imageSrcFor = (name: string) => {
    return animalAssetUrl(name);
  };

  const animalCard = (a: Animal, target: 'h1' | 'h2') => {
    const src = target === 'h1' ? h1 : h2;
    const disabled = usedIds.has(a.ID) && !src.animalIds.includes(a.ID);
    const selected = src.animalIds.includes(a.ID);
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
            <div style={{ fontSize: 12 }}>{a.skill_name} (Cost {a.skill_cost})</div>
            {(() => {
              const isPicked = src.animalIds.includes(a.ID);
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
                        checked={src.selectedAnimalId === a.ID}
                        onChange={() => {
                          const setter = target === 'h1' ? setH1 : setH2;
                          setter({ ...src, selectedAnimalId: a.ID });
                        }}
                        disabled={!selected}
                      />
                      <span>Use this animal's special ability</span>
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
    <div className="animals-grid">
      {animals.map((a) => animalCard(a, target))}
    </div>
  );

  return (
    <div style={{ border: '1px solid #333', padding: 16, borderRadius: 8 }}>
      <h3>Create Your Hybrids</h3>
      <div className="hybrid-creation-grid">
        <section>
          <h4>Hybrid 1</h4>
          {grid('h1')}
          <div style={{ fontSize: 12, marginTop: 4 }}>Pick 2 to 3 animals and choose 1 special ability among them</div>
          <div style={{ fontSize: 12, marginTop: 4, color: '#ccc' }}>Name (auto): {h1Name || '-'}</div>
        </section>
        <section>
          <h4>Hybrid 2</h4>
          {grid('h2')}
          <div style={{ fontSize: 12, marginTop: 4 }}>Pick 2 to 3 animals (no overlap with Hybrid 1) and choose 1 special ability</div>
          <div style={{ fontSize: 12, marginTop: 4, color: '#ccc' }}>Name (auto): {h2Name || '-'}</div>
        </section>
        <button onClick={handleSubmit} disabled={!isValidSelection || submitting} style={{ padding: '10px 16px' }}>
          {submitting ? 'Creating…' : 'Create Hybrids'}
        </button>
        <div style={{ fontSize: 12, color: '#aaa' }}>* Names are generated automatically by the game.</div>
      </div>
    </div>
  );
};

export default HybridCreation;
