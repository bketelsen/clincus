import { api } from '../lib/api';
import type { Session } from '../lib/types';

const store = $state<{ items: Session[] }>({ items: [] });

export function getSessions(): Session[] {
  return store.items;
}

export async function loadSessions() {
  store.items = await api.listSessions();
}

export function addSession(s: Session) {
  store.items = [...store.items, s];
}

export function removeSession(id: string) {
  store.items = store.items.filter(s => s.id !== id);
}
