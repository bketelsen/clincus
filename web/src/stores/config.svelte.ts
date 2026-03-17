import { api } from '../lib/api';
import type { ClincusConfig } from '../lib/types';

const store = $state<{ config: ClincusConfig | null; loading: boolean; error: string | null }>({
  config: null,
  loading: false,
  error: null,
});

export function getConfig(): ClincusConfig | null {
  return store.config;
}

export function getConfigLoading(): boolean {
  return store.loading;
}

export function getConfigError(): string | null {
  return store.error;
}

export async function loadConfig() {
  store.loading = true;
  store.error = null;
  try {
    store.config = await api.getConfig();
  } catch (e) {
    store.error = e instanceof Error ? e.message : 'Failed to load config';
  } finally {
    store.loading = false;
  }
}
