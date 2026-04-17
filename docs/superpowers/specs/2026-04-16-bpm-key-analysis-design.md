# BPM & Musical Key Analysis — Design

**Date:** 2026-04-16
**Status:** Draft
**Feature #:** 1 of the DJ-productivity roadmap

## Problem

CrateDrop stores and plays tracks, but has no tempo or key metadata. DJs cannot:

- Build crates filtered by BPM range
- Identify harmonically compatible tracks
- Sort a library by tempo for set construction

Every other planned DJ feature (smart crates, "plays well next", Rekordbox export) depends on having BPM and musical key.

## Goal

Every track in the library has a detected BPM and musical key (Camelot notation), viewable in the track list. Detection runs asynchronously without blocking uploads or the HTTP API. Users can override wrong detections.

## Non-Goals (v1)

- Energy, loudness, danceability, or other audio features
- Bulk edit UI
- "Re-analyze" button (manual edit is the escape hatch)
- Filter bar by BPM range or key compatibility (feature #4: smart crates)
- Harmonic mixing suggestions (feature #5)
- Rekordbox/Serato export (feature #2)
- Waveform visualization (feature #3)

## Key Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Analysis engine | essentia `streaming_extractor_music` | Single CLI, JSON output, fits existing FFprobe subprocess pattern. No Python runtime needed. |
| Scope | Backfill entire library + analyze every new upload | Whole library becomes DJ-ready immediately. |
| Manual edit | Editable with confidence shown | ~5–10% of detections are wrong; users need an escape hatch. Confidence dot helps them know which to review. |
| UI surface | Track list columns only (v1) | Two new columns (BPM, Key), sortable, inline-editable. Filter bar and detail view deferred. |
| Worker model | In-process goroutine, DB-polled, concurrency=1 | Matches the existing SoundCloud ticker pattern. Crash-safe via SQLite atomic updates. One analysis at a time keeps the Pi responsive for HTTP traffic. |

## Architecture

A new `backend/analysis/` module, structured to match the existing `backend/soundcloud/` layout:

```
backend/analysis/
  essentia.go    # subprocess wrapper: Analyze(ctx, path) (Result, error)
  camelot.go     # pure mapping: ToCamelot(key, scale) string
  ticker.go      # background loop, started from server boot
  manager.go     # DB claim + update logic
  repository.go  # SQL for reading/writing analysis fields
```

The ticker runs inside the main Go server process, started from `server/` alongside the SoundCloud ticker. On each 10-second tick:

1. `SELECT id, file_path FROM tracks WHERE analysis_status='pending' AND (next_retry_at IS NULL OR next_retry_at <= datetime('now')) ORDER BY uploaded_at LIMIT 1`
2. Shell out to `streaming_extractor_music <input> <output_json>`
3. Parse JSON, map key+scale to Camelot
4. `UPDATE tracks SET bpm=?, bpm_confidence=?, musical_key=?, key_confidence=?, analyzed_at=datetime('now'), analysis_status='analyzed' WHERE id=?`

On failure, the track remains `analysis_status='pending'` with `analysis_retry_count` incremented and `next_retry_at` set via exponential backoff (10m, 1h, 6h). After 3 failed attempts the status flips to `failed` and the worker stops retrying it. This keeps the ticker query simple (one status for "eligible work") while still surfacing terminal failures in the UI.

## Data Model

Migration: `backend/internal/db/migrations/003_add_track_analysis.sql`

```sql
ALTER TABLE tracks ADD COLUMN bpm REAL;
ALTER TABLE tracks ADD COLUMN bpm_confidence REAL;
ALTER TABLE tracks ADD COLUMN musical_key TEXT;              -- Camelot, e.g. "8A"
ALTER TABLE tracks ADD COLUMN key_confidence REAL;
ALTER TABLE tracks ADD COLUMN analyzed_at TIMESTAMP;
ALTER TABLE tracks ADD COLUMN analysis_status TEXT NOT NULL DEFAULT 'pending';
ALTER TABLE tracks ADD COLUMN analysis_error TEXT;
ALTER TABLE tracks ADD COLUMN analysis_retry_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tracks ADD COLUMN next_retry_at TIMESTAMP;
CREATE INDEX idx_tracks_analysis_status ON tracks(analysis_status, next_retry_at);
```

`analysis_status` values:
- `pending` — not yet analyzed (default for new and backfilled rows)
- `analyzed` — detection succeeded
- `failed` — detection failed (retries exhausted or non-retryable, e.g. file missing)
- `user_edited` — user manually set BPM or key; terminal state, not touched by the worker

Backfill is automatic: because the default is `pending`, existing rows become eligible for analysis as soon as the migration runs.

## Components

### `backend/analysis/essentia.go`

```go
type Result struct {
    BPM           float64
    BPMConfidence float64
    Key           string // Camelot, e.g. "8A"
    KeyConfidence float64
}

func Analyze(ctx context.Context, audioPath string) (Result, error)
```

- Uses `exec.CommandContext` with a 90-second timeout.
- Writes essentia output to a temp JSON file, reads and parses, deletes the temp file.
- Ignores fields other than BPM, key, scale, and the respective confidence scores.
- Returns a typed error (`ErrBinaryMissing`, `ErrTimeout`, `ErrMalformedOutput`, `ErrFileMissing`) so the worker can decide whether to retry.

### `backend/analysis/camelot.go`

Pure function. All 24 mappings covered by unit tests.

```go
func ToCamelot(key, scale string) string
// ("A", "minor") -> "8A"
// ("C", "major") -> "8B"
// etc.
```

### `backend/analysis/ticker.go`

```go
func Start(ctx context.Context, mgr *Manager, interval time.Duration)
```

Mirrors `backend/soundcloud/ticker.go`. Started from `server/` with a 10s interval.

### `backend/analysis/manager.go`

Orchestrates: claim next track from repo, call `essentia.Analyze`, write result back. Does not log at info level per track (would be noisy during backfill); uses debug. Failures logged at warn with track id + error.

### HTTP API

New endpoint in `backend/tracks/`:

```
PATCH /api/tracks/:id
Body: { "bpm"?: number, "musical_key"?: string }
```

- Validates: `bpm` in `[50, 250]`, `musical_key` matches `^(1[0-2]|[1-9])[AB]$`.
- On success: updates the specified fields, sets `analysis_status='user_edited'`, clears `analysis_error`.
- Auth: reuses the existing track-owner or admin guard.

### Frontend

Track list table gains two columns: **BPM** and **Key**. Both sortable. Inline-editable on double-click (text input with client-side validation matching the server rules). Confidence visualized as a colored dot beside the value:

- green: confidence > 0.7
- amber: 0.4–0.7
- red: < 0.4
- grey outline: `pending`
- red outline: `failed`
- no dot: `user_edited`

## Data Flow

```
Upload      -> tracks row inserted, analysis_status='pending'
Ticker tick -> claims row, runs essentia (15-30s on Pi 4)
            -> success: writes bpm/key/confidence, status='analyzed'
            -> failure: increments retry_count, sets next_retry_at
Frontend    -> React Query refetch picks up new values on next poll or window focus
User edit   -> PATCH /api/tracks/:id -> status='user_edited', worker ignores
```

## Error Handling

| Condition | Behavior |
|---|---|
| essentia binary not on PATH at startup | Log loudly at boot, skip ticker start, server still runs, tracks stay `pending` |
| essentia timeout (>90s) | Stay `pending`, retry with backoff, flip to `failed` after 3 attempts |
| Malformed JSON output | Stay `pending`, retry with backoff, flip to `failed` after 3 attempts |
| Audio file missing on disk | `failed` immediately, no retry (non-recoverable) |
| Unknown key returned by essentia | Store BPM with `analyzed` status; `musical_key` left NULL, `key_confidence=0` |
| DB write fails | Ticker logs and moves on; next tick retries naturally |

Retry backoff: 10 minutes, 1 hour, 6 hours. After 3 failures the track is terminal-`failed` and must be user-edited or re-uploaded.

## Testing

**Unit tests** (`camelot_test.go`, `essentia_test.go`, `manager_test.go`):

- All 24 `ToCamelot` mappings, plus invalid inputs.
- `essentia.Analyze` JSON parsing against fixture files at `testdata/essentia_*.json`.
- Manager retry/backoff state machine using a mock clock and a stub essentia implementation.

**Integration tests** (`ticker_integration_test.go`):

- Stub `streaming_extractor_music` as a bash script on PATH that echoes a fixture.
- In-memory SQLite, seed a `pending` track, run one tick, assert row updated.
- Seed a `failed` track with past `next_retry_at`, run tick, assert it retries.
- Seed a `user_edited` track, run tick, assert it is untouched.

**E2E** (Playwright):

- Upload a known fixture track, wait up to 60s, assert the track list renders expected BPM (±2) and key.
- Double-click the BPM cell, edit, save, assert the value persists and the confidence dot is gone.

**Target coverage:** 80%+ on the `backend/analysis/` package.

## Deploy

essentia's `streaming_extractor_music` must be on PATH on the Pi.

- On Debian Bookworm (Pi OS base): `apt install essentia-examples` provides the binary.
- Verify at deploy time: `which streaming_extractor_music` in `deploy-local.sh` / `deploy-prod.sh`.
- If unavailable via apt for the target Pi OS version, document a manual build from source in `README.md` as a deploy prereq.

The server must run even if the binary is missing; it just won't analyze.

## Open Questions

None blocking. Known small decisions to confirm during implementation:

- Exact ticker interval (10s is a starting guess; can tune based on Pi load)
- Retry backoff values (10m/1h/6h is a starting guess)
- Frontend confidence dot thresholds (0.4/0.7 is a starting guess)
