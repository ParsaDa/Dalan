import { useSyncExternalStore } from 'react';

const KEY_SIMPLE = 'dalan-simple-mode';
const KEY_SLIM = 'dalan-rail-slim';
const listeners = new Set<() => void>();

function read(key: string): boolean {
  try {
    return localStorage.getItem(key) === 'true';
  } catch {
    return false;
  }
}

let state = { simple: read(KEY_SIMPLE), slim: read(KEY_SLIM) };

function set(next: Partial<typeof state>) {
  state = { ...state, ...next };
  try {
    localStorage.setItem(KEY_SIMPLE, String(state.simple));
    localStorage.setItem(KEY_SLIM, String(state.slim));
  } catch {
    // Private mode: the choice lasts for this tab only.
  }
  listeners.forEach((l) => l());
}

export const shellPrefs = {
  setSimple: (simple: boolean) => set({ simple }),
  setSlim: (slim: boolean) => set({ slim }),
};

export function useShellPrefs() {
  return useSyncExternalStore(
    (l) => {
      listeners.add(l);
      return () => listeners.delete(l);
    },
    () => state,
  );
}
