# Marketplace Plan

## Vision

A browsable, searchable marketplace for agent plugins and profiles. Hosted on
the existing registry server — any deployment can point to a public registry
(community packages) and/or a private one (org-internal packages).

```
┌──────────────────────────────────────────────────┐
│  Agent Config (pluginSources)                     │
│                                                    │
│  - name: community                                │
│    type: registry                                  │
│    url: https://registry.bitop.dev                 │
│                                                    │
│  - name: internal                                  │
│    type: registry                                  │
│    url: http://registry.corp.internal:9080          │
│    publishToken: corp-token-abc                    │
└──────────────────────────────────────────────────┘
         │                        │
         ▼                        ▼
┌─────────────────┐    ┌─────────────────┐
│ Public Registry  │    │ Private Registry │
│ (community)      │    │ (org-internal)   │
│ read: anyone     │    │ read: VPN only   │
│ publish: tokens  │    │ publish: tokens  │
└─────────────────┘    └─────────────────┘
```

Workers and CLI already support multiple sources. `agent plugins search` queries
all configured sources. `agent plugins install foo --source community` picks a
specific one. This plan focuses on making the public registry worth visiting.

---

## What we're building

### Registry server enhancements (agent-registry)

1. **Download counts**
   - Increment on artifact download (`GET /artifacts/...`)
   - Store in a `package_stats` table (or in-memory + periodic flush)
   - Expose in index: `downloads` field on each package/profile
   - Sort search results by downloads (popularity)

2. **Publisher attribution**
   - Publish tokens get a `publisher` name (e.g. "bitop-dev", "nick")
   - Store publisher on each package version
   - Show in index and detail responses
   - Future: map to GitHub usernames

3. **Search improvements**
   - `GET /v1/search?q=grafana` — full-text search across name, description, keywords, tools
   - `GET /v1/search?category=integration` — filter by category
   - `GET /v1/search?runtime=mcp` — filter by runtime type
   - `GET /v1/search?sort=downloads` — sort by popularity
   - Returns enriched data (tools, capabilities, downloads, publisher)

4. **Package detail endpoint**
   - `GET /v1/packages/{name}/detail.json` — full manifest data
   - Returns: all versions, tools, config schema, dependencies, README
   - `GET /v1/profiles/{name}/detail.json` — same for profiles
   - Returns: tools, model, capabilities, accepts/returns, instructions summary

5. **README support**
   - Extract `README.md` from tarballs at publish time
   - Serve via detail endpoint as rendered HTML or raw markdown
   - Display in marketplace UI

6. **Publish tokens with publisher identity**
   - `POST /v1/auth/tokens` — admin creates named publish tokens
   - Each token has: name, publisher, scopes (publish, admin)
   - Publisher name attached to packages at publish time

### Marketplace web UI (embedded in registry)

A public-facing web UI served by the registry at `/` — separate from the
gateway dashboard. This is what people visit to discover plugins and profiles.

**Pages:**

```
/                       — Landing: featured, popular, recent
/plugins                — Browse all plugins
/plugins/{name}         — Plugin detail (versions, tools, README, install cmd)
/profiles               — Browse all profiles
/profiles/{name}        — Profile detail (tools, model, capabilities, README)
/search?q=...           — Search results
```

**Landing page:**
- Hero: "Agent Plugins & Profiles"
- Search bar
- Popular plugins (by downloads)
- Recently published
- Category quick links (integration, orchestration, monitoring, etc)

**Plugin detail page:**
- Name, version, publisher, downloads
- Description + README (markdown rendered)
- Tools contributed (list with descriptions)
- Config schema (what needs configuring)
- Dependencies
- Install command: `agent plugins install {name}`
- Version history

**Profile detail page:**
- Name, version, publisher, downloads
- Description + README
- Model, provider, capabilities
- Tools used (with links to plugin pages)
- Accepts/returns
- Extends (parent profile link)
- Install command: `agent profiles install {name}`

**Tech stack:**
- Same as gateway dashboard: Svelte 5 + Vite + Bun + Tailwind + shadcn-svelte
- Built to `cmd/registry-server/web/` and embedded via `go:embed`
- One binary serves API + marketplace UI

### Gateway dashboard integration

The gateway dashboard gets a "Marketplace" section that links to / embeds
content from the configured registry:

- **Browse tab** on Plugins page — shows registry plugins with install counts
- **Browse tab** on Agents page — shows registry profiles
- **Search bar** that queries the registry search endpoint
- **"Install" action** — not directly (workers do on-demand), but shows
  the install command for CLI usage

### CLI enhancements

- `agent plugins search` already works — will now show downloads + publisher
- `agent plugins info {name}` — show full detail from registry
- `agent profiles search` — search profiles (new command)
- `agent profiles info {name}` — show full profile detail

---

## Data model additions

### package_stats table
```sql
CREATE TABLE IF NOT EXISTS package_stats (
  name          TEXT NOT NULL,
  type          TEXT NOT NULL DEFAULT 'plugin',  -- 'plugin' or 'profile'
  downloads     INTEGER DEFAULT 0,
  last_download TIMESTAMPTZ,
  PRIMARY KEY (name, type)
);
```

### publish_tokens table
```sql
CREATE TABLE IF NOT EXISTS publish_tokens (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  token_hash  TEXT NOT NULL UNIQUE,
  publisher   TEXT NOT NULL,
  scopes      TEXT[] DEFAULT '{"publish"}',
  created_at  TIMESTAMPTZ DEFAULT now(),
  last_used   TIMESTAMPTZ
);
```

### Package metadata additions
```json
{
  "name": "ddg-research",
  "version": "0.1.0",
  "publisher": "bitop-dev",
  "downloads": 142,
  "publishedAt": "2026-03-21T12:00:00Z",
  "readme": "# DDG Research\n\nReal web search via DuckDuckGo...",
  "tools": ["ddg/search", "ddg/fetch"],
  "configSchema": { ... }
}
```

---

## Implementation phases

### Phase 1: Registry data layer
- Download counting on artifact requests
- Publisher identity on publish tokens
- Search endpoint with filters and sorting
- Package/profile detail endpoints
- README extraction from tarballs
- SQLite or PostgreSQL for stats (registry currently uses filesystem)

**Decision needed:** The registry currently uses the filesystem for everything.
For download counts and tokens, we need either:
- A) SQLite embedded (simple, single-node)
- B) PostgreSQL (already have it for gateway)
- C) JSON files on disk (simplest, no DB)

Recommendation: **SQLite** — keeps the registry self-contained (no external DB
dependency), but gives us real queries. The registry is a single replica anyway.

### Phase 2: Marketplace web UI
- Svelte project in `registry/marketplace/`
- Landing page with search + popular + recent
- Plugin browse + detail pages
- Profile browse + detail pages
- Markdown rendering for READMEs
- Build → `cmd/registry-server/web/` → go:embed

### Phase 3: Gateway + CLI integration
- Gateway dashboard "Marketplace" tab (or embeds registry search)
- CLI `plugins info` and `profiles search` commands
- Download count display in search results

### Phase 4: Polish
- Category pages
- Version comparison
- "Used by" profiles (which profiles use this plugin)
- Dependency graph visualization
- Social proof (download badges, "popular" tags)

---

## Open decisions

1. **Registry storage backend** — SQLite (recommended) vs PostgreSQL vs flat files
2. **README format** — markdown only, or also support plain text?
3. **Search implementation** — SQLite FTS5 vs simple LIKE queries vs in-memory
4. **Marketplace URL** — same domain as registry API, or separate?
   Recommendation: same origin, `/` = UI, `/v1/` = API (like gateway)
5. **Rate limiting** — needed for public registry? Not yet, add later.
6. **Package size limits** — max tarball size? 50MB reasonable default.
7. **Naming conventions** — should plugins be scoped? e.g. `@bitop/ddg-research`
   Not yet, but plan for it. Add optional `publisher` prefix later.

---

## What we're NOT building (yet)

- User accounts / login / OAuth
- Ratings and reviews
- Automated testing of published packages
- Package signing / verification
- Billing / paid plugins
- Multi-region registry replication
- Web-based plugin/profile editor

These are all "marketplace v2" features. Get the basics right first.

---

## Estimated effort

| Phase | Scope | Effort |
|-------|-------|--------|
| Phase 1 | Registry data layer | Medium (SQLite, search, detail endpoints, README) |
| Phase 2 | Marketplace UI | Medium (Svelte, 5-6 pages, markdown rendering) |
| Phase 3 | Gateway + CLI integration | Small (API calls, display tweaks) |
| Phase 4 | Polish | Small-Medium (categories, badges, dep graph) |

Total: ~2-3 sessions of focused work.
