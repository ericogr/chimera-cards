// Lightweight runtime config helpers used across the frontend.

export function getRuntimeConfig(): Record<string, any> {
  return (window as any)._env_ || {};
}

export function getMissingRuntimeKeys(required: string[]): string[] {
  const cfg = getRuntimeConfig();
  return required.filter(k => !(cfg[k] && String(cfg[k]).trim() !== ''));
}

export function safeSetLocal(key: string, value: string): void {
  try {
    localStorage.setItem(key, value);
  } catch {}
}

export function safeRemoveLocal(key: string): void {
  try {
    localStorage.removeItem(key);
  } catch {}
}

