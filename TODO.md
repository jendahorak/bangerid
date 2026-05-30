# Bangerid TODO

## Phase 1 — Core ✅
- [x] Spotify OAuth2 flow
- [x] Fetch liked songs, in-memory cache
- [x] Grid rendering with Go templates
- [x] Playback via Web Playback SDK + `/play` endpoint
- [x] Lazy loading (`loading="lazy"`)

## Phase 2 — Datastar Migration
- [ ] Swap HTMX script tag for Datastar
- [ ] `/grid`: return SSE via `datastar.NewSSE` + `sse.PatchElements`
- [ ] `/play`: use `datastar.ReadSignals` instead of query params/form values
- [ ] Declare signals: `currentTrack`, `isPaused`, `deviceId`
- [ ] Bridge Spotify SDK events (`ready`, `player_state_changed`) into signals
- [ ] Replace `app.js` DOM manipulation with `data-class`, `data-show`
- [ ] Replace prev/next button JS with `data-on:click` + signal-based index tracking
- [ ] Loading states via `data-show` instead of `htmx:afterRequest`

## Phase 3 — Persistence
- [ ] Store track metadata to filesystem (JSON file in `data/`)
- [ ] Load from disk on startup, skip Spotify fetch if fresh
- [ ] Staleness check: re-fetch if metadata > 24h old
- [ ] Manual "Refresh" button to force re-fetch

## Phase 4 — Image Proxy + BlurHash
- [ ] `/img-proxy/:id` endpoint — fetch 64x64 from Spotify, convert to WebP, write to `data/images/`
- [ ] Serve from disk on cache hit with `Cache-Control: immutable`
- [ ] Generate BlurHash string on first fetch, store with track metadata
- [ ] Render BlurHash to `<canvas>` placeholder via `data-effect`
- [ ] Fade transition: `data-on:load` flips signal → `data-class` reveals real image

## Phase 5 — Progressive Loading
- [ ] Stream grid in batches (50 cards per SSE event) via `sse.PatchElements` with `WithModeAppend`
- [ ] HTTP/2 for multiplexed image requests
- [ ] Aggressive `Cache-Control` on all static assets
