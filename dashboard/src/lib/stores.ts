// Global reactive stores

import { writable } from "svelte/store";

// API key (persisted to localStorage)
function createApiKeyStore() {
  const stored = localStorage.getItem("agent-api-key") || "";
  const { subscribe, set } = writable(stored);
  return {
    subscribe,
    set: (value: string) => {
      localStorage.setItem("agent-api-key", value);
      set(value);
    },
  };
}

export const apiKey = createApiKeyStore();

// Events buffer (last 50)
export const events = writable<
  Array<{ type: string; data: Record<string, unknown>; timestamp: string }>
>([]);

export function addEvent(event: {
  type: string;
  data: Record<string, unknown>;
  timestamp: string;
}) {
  events.update((e) => {
    const next = [event, ...e];
    return next.slice(0, 50);
  });
}
