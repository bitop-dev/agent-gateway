// Minimal hash-based SPA router
// Uses #/path style to work with go:embed static serving

import { writable, derived } from "svelte/store";

export const hash = writable(window.location.hash.slice(1) || "/");

// Listen for hash changes
window.addEventListener("hashchange", () => {
  hash.set(window.location.hash.slice(1) || "/");
});

export function navigate(path: string) {
  window.location.hash = path;
}

// Parse route params from patterns like "/tasks/:id"
export function matchRoute(
  pattern: string,
  path: string
): Record<string, string> | null {
  const patternParts = pattern.split("/");
  const pathParts = path.split("/");

  if (patternParts.length !== pathParts.length) return null;

  const params: Record<string, string> = {};
  for (let i = 0; i < patternParts.length; i++) {
    if (patternParts[i].startsWith(":")) {
      params[patternParts[i].slice(1)] = pathParts[i];
    } else if (patternParts[i] !== pathParts[i]) {
      return null;
    }
  }
  return params;
}

// Derived store for the current route segment
export const currentPath = derived(hash, ($hash) => $hash.split("?")[0]);
