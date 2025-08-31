export function animalKeyFromNames(names: string[]): string {
  const parts = (names || [])
    .map((n) => (n || '').toString().trim().toLowerCase().replace(/\s+/g, '_'))
    .filter(Boolean)
    .sort();
  return parts.join('_');
}

export function animalAssetUrl(name: string): string {
  const file = name.trim().toLowerCase().replace(/\s+/g, '_') + '.png';
  return `/api/assets/animals/${file}`;
}

export function hybridAssetUrlFromNames(names: string[]): string {
  const k = animalKeyFromNames(names);
  return k ? `/api/assets/hybrids/${k}.png` : '';
}

