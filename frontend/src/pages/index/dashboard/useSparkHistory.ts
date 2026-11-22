import { useEffect, useSyncExternalStore } from 'react';

const POINTS = 24;
const MIN_GAP_MS = 4000;
const EMPTY: number[] = [];
const store = new Map<string, { at: number; values: number[] }>();
const listeners = new Set<() => void>();

// One sample per refresh; history survives switching node tabs within the session.
export function useSparkHistory(key: string, value: number): number[] {
  useEffect(() => {
    if (!Number.isFinite(value)) return;
    const prev = store.get(key) ?? { at: 0, values: [] };
    const now = Date.now();
    if (now - prev.at < MIN_GAP_MS && prev.values.length) return;
    store.set(key, { at: now, values: [...prev.values, value].slice(-POINTS) });
    listeners.forEach((l) => l());
  }, [key, value]);
  return useSyncExternalStore(
    (l) => {
      listeners.add(l);
      return () => listeners.delete(l);
    },
    () => store.get(key)?.values ?? EMPTY,
  );
}
