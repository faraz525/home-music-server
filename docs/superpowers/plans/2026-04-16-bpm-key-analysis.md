# BPM & Musical Key Analysis Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every track in the library gets auto-detected BPM and musical key (Camelot notation), displayed in the track list as sortable columns, inline-editable with confidence indicators.

**Architecture:** A new `backend/analysis/` Go package running an in-process goroutine that polls SQLite every 10s for `pending` tracks, shells out to essentia's `streaming_extractor_music`, parses JSON output, and writes BPM + Camelot key back. Failed analyses retry with exponential backoff (10m, 1h, 6h) before becoming terminal `failed`. User edits set a `user_edited` status the worker respects. Frontend `LibraryPage` gains two new columns (BPM, Key) with confidence dots and inline editing.

**Tech Stack:** Go 1.23 + Gin + SQLite (`mattn/go-sqlite3`), React 18 + Vite + TanStack Query + Tailwind, essentia `streaming_extractor_music` CLI on PATH. Tests use Go's stdlib `testing` package with in-memory SQLite (`:memory:`).

**Spec:** [`docs/superpowers/specs/2026-04-16-bpm-key-analysis-design.md`](../specs/2026-04-16-bpm-key-analysis-design.md)

---

## File Structure

### New files
- `backend/internal/db/migrations/003_add_track_analysis.sql` — schema changes
- `backend/analysis/camelot.go` — pure `(key, scale) -> Camelot` mapping
- `backend/analysis/camelot_test.go`
- `backend/analysis/essentia.go` — subprocess wrapper
- `backend/analysis/essentia_test.go`
- `backend/analysis/essentia_parse.go` — split so the JSON parser is testable without a subprocess
- `backend/analysis/essentia_parse_test.go`
- `backend/analysis/testdata/essentia_success.json` — fixture
- `backend/analysis/repository.go` — DB claim/update helpers
- `backend/analysis/repository_test.go`
- `backend/analysis/manager.go` — orchestrates one claim + analyze + write
- `backend/analysis/manager_test.go`
- `backend/analysis/ticker.go` — `StartLoop(ctx, *Manager)` background poller
- `backend/tracks/patch_handler.go` — new `PATCH /api/tracks/:id` handler
- `backend/tracks/patch_handler_test.go`

### Modified files
- `backend/internal/db/db.go` — apply migration 003 on boot, detect missing `bpm` column
- `backend/internal/models/models.go` — extend `Track` struct with new optional fields
- `backend/tracks/tracksdata.go` — include new columns in all SELECT/INSERT/UPDATE statements
- `backend/tracks/routes.go` — register PATCH route
- `backend/tracks/tracksmanager.go` — add `UpdateAnalysis(ctx, trackID, bpm *float64, key *string)` method for user overrides
- `backend/main.go` — init `analysis.Manager` + `analysis.Repository`, start ticker goroutine
- `frontend/src/types/crates.ts` — (this file currently holds `TrackList`) extend `Track` type with new fields
- `frontend/src/pages/LibraryPage.tsx` — add BPM/Key columns and inline editing
- `frontend/src/lib/api.ts` — add `tracksApi.patch(id, payload)`
- `frontend/src/hooks/useQueries.ts` — add `useUpdateTrackAnalysis()` mutation
- `deploy-local.sh`, `deploy-prod.sh` — add `which streaming_extractor_music` verification
- `README.md` — deploy prereq note

---

## Task 1: DB migration and model extension

**Files:**
- Create: `backend/internal/db/migrations/003_add_track_analysis.sql`
- Modify: `backend/internal/db/db.go` (end of `migrate()` function, after the 002 block around line 117)
- Modify: `backend/internal/models/models.go:15-32` (Track struct)

- [ ] **Step 1: Create the migration file**

Create `backend/internal/db/migrations/003_add_track_analysis.sql`:

```sql
-- Track analysis fields for BPM + Musical Key detection
ALTER TABLE tracks ADD COLUMN bpm REAL;
ALTER TABLE tracks ADD COLUMN bpm_confidence REAL;
ALTER TABLE tracks ADD COLUMN musical_key TEXT;              -- Camelot notation, e.g. "8A"
ALTER TABLE tracks ADD COLUMN key_confidence REAL;
ALTER TABLE tracks ADD COLUMN analyzed_at DATETIME;
ALTER TABLE tracks ADD COLUMN analysis_status TEXT NOT NULL DEFAULT 'pending';
ALTER TABLE tracks ADD COLUMN analysis_error TEXT;
ALTER TABLE tracks ADD COLUMN analysis_retry_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tracks ADD COLUMN next_retry_at DATETIME;

CREATE INDEX IF NOT EXISTS idx_tracks_analysis_status
    ON tracks(analysis_status, next_retry_at);
```

Also append the same columns to `backend/internal/db/migrations/schema.sql` inside the `CREATE TABLE tracks (...)` block (between `bitrate INTEGER,` and `file_path TEXT NOT NULL,`) so fresh DBs include them. The matching index addition goes at the end of the tracks-related section:

```sql
    bitrate INTEGER,
    bpm REAL,
    bpm_confidence REAL,
    musical_key TEXT,
    key_confidence REAL,
    analyzed_at DATETIME,
    analysis_status TEXT NOT NULL DEFAULT 'pending',
    analysis_error TEXT,
    analysis_retry_count INTEGER NOT NULL DEFAULT 0,
    next_retry_at DATETIME,
    file_path TEXT NOT NULL,
```

And add this index near the other track indexes:

```sql
CREATE INDEX IF NOT EXISTS idx_tracks_analysis_status
    ON tracks(analysis_status, next_retry_at);
```

- [ ] **Step 2: Wire the migration into `db.go`**

At the end of `migrate()` in `backend/internal/db/db.go` (before the final `return nil`), add:

```go
// Check if bpm column exists on tracks table
var bpmColCount int
_ = d.QueryRow(`
    SELECT COUNT(*)
    FROM pragma_table_info('tracks')
    WHERE name='bpm'
`).Scan(&bpmColCount)
if bpmColCount == 0 {
    migrationSQL, err := migrationsFS.ReadFile("migrations/003_add_track_analysis.sql")
    if err != nil {
        return fmt.Errorf("failed to read migration 003_add_track_analysis: %w", err)
    }
    if _, err := d.Exec(string(migrationSQL)); err != nil {
        return fmt.Errorf("failed to execute migration 003_add_track_analysis: %w", err)
    }
}
```

- [ ] **Step 3: Extend the Track model**

In `backend/internal/models/models.go`, replace the `Track` struct with:

```go
type Track struct {
    ID                 string     `json:"id"`
    OwnerUserID        string     `json:"owner_user_id"`
    OriginalFilename   string     `json:"original_filename"`
    ContentType        string     `json:"content_type"`
    SizeBytes          int64      `json:"size_bytes"`
    DurationSeconds    *float64   `json:"duration_seconds,omitempty"`
    Title              *string    `json:"title,omitempty"`
    Artist             *string    `json:"artist,omitempty"`
    Album              *string    `json:"album,omitempty"`
    Genre              *string    `json:"genre,omitempty"`
    Year               *int       `json:"year,omitempty"`
    SampleRate         *int       `json:"sample_rate,omitempty"`
    Bitrate            *int       `json:"bitrate,omitempty"`
    BPM                *float64   `json:"bpm,omitempty"`
    BPMConfidence      *float64   `json:"bpm_confidence,omitempty"`
    MusicalKey         *string    `json:"musical_key,omitempty"`
    KeyConfidence      *float64   `json:"key_confidence,omitempty"`
    AnalyzedAt         *time.Time `json:"analyzed_at,omitempty"`
    AnalysisStatus     string     `json:"analysis_status"`
    FilePath           string     `json:"file_path"`
    CreatedAt          time.Time  `json:"created_at"`
    UpdatedAt          time.Time  `json:"updated_at"`
}
```

Note `AnalysisStatus` is non-pointer `string` because the migration default ensures it's always present. We do NOT expose `analysis_error`, `analysis_retry_count`, or `next_retry_at` via JSON — internal only.

- [ ] **Step 4: Update the tracks repository to read/write new columns**

In `backend/tracks/tracksdata.go`, every SELECT and INSERT involving `tracks` must include the new columns. For each `SELECT id, owner_user_id, original_filename, content_type, size_bytes, duration_seconds, title, artist, album, genre, year, sample_rate, bitrate, file_path, created_at, updated_at FROM tracks ...` replace with:

```sql
SELECT id, owner_user_id, original_filename, content_type, size_bytes,
    duration_seconds, title, artist, album, genre, year, sample_rate, bitrate,
    bpm, bpm_confidence, musical_key, key_confidence, analyzed_at, analysis_status,
    file_path, created_at, updated_at
FROM tracks ...
```

In `CreateTrack`, leave the INSERT column list alone — the new columns default via the migration. But the row scans must be extended. Add helper scan logic at the top of the file (or wherever scan is done):

```go
func scanTrack(row interface{ Scan(dest ...any) error }) (*imodels.Track, error) {
    var t imodels.Track
    var duration, bpm, bpmConf, keyConf sql.NullFloat64
    var title, artist, album, genre, key sql.NullString
    var year, sampleRate, bitrate sql.NullInt64
    var analyzedAt sql.NullTime

    err := row.Scan(
        &t.ID, &t.OwnerUserID, &t.OriginalFilename, &t.ContentType, &t.SizeBytes,
        &duration, &title, &artist, &album, &genre, &year, &sampleRate, &bitrate,
        &bpm, &bpmConf, &key, &keyConf, &analyzedAt, &t.AnalysisStatus,
        &t.FilePath, &t.CreatedAt, &t.UpdatedAt,
    )
    if err != nil {
        return nil, err
    }
    if duration.Valid {
        t.DurationSeconds = &duration.Float64
    }
    if title.Valid {
        v := title.String; t.Title = &v
    }
    if artist.Valid {
        v := artist.String; t.Artist = &v
    }
    if album.Valid {
        v := album.String; t.Album = &v
    }
    if genre.Valid {
        v := genre.String; t.Genre = &v
    }
    if year.Valid {
        v := int(year.Int64); t.Year = &v
    }
    if sampleRate.Valid {
        v := int(sampleRate.Int64); t.SampleRate = &v
    }
    if bitrate.Valid {
        v := int(bitrate.Int64); t.Bitrate = &v
    }
    if bpm.Valid {
        t.BPM = &bpm.Float64
    }
    if bpmConf.Valid {
        t.BPMConfidence = &bpmConf.Float64
    }
    if key.Valid {
        v := key.String; t.MusicalKey = &v
    }
    if keyConf.Valid {
        t.KeyConfidence = &keyConf.Float64
    }
    if analyzedAt.Valid {
        t.AnalyzedAt = &analyzedAt.Time
    }
    return &t, nil
}
```

Replace existing scan loops (`GetTracks`, `GetAllTracks`, `GetTrackByID`, etc.) to use this helper and the extended SELECT. Add `import "database/sql"` if not present.

- [ ] **Step 5: Build and run**

Run from `backend/`:

```bash
go build ./...
```

Expected: clean build with no errors.

Start the server against a scratch data dir and verify the migration applies:

```bash
rm -rf /tmp/cratedrop-test-data
DATA_DIR=/tmp/cratedrop-test-data PORT=8081 go run . &
SERVER_PID=$!
sleep 2
sqlite3 /tmp/cratedrop-test-data/db/cratedrop.sqlite "PRAGMA table_info(tracks);" | grep -E "bpm|musical_key|analysis_status"
kill $SERVER_PID
```

Expected output shows `bpm`, `bpm_confidence`, `musical_key`, `key_confidence`, `analyzed_at`, `analysis_status`, `analysis_error`, `analysis_retry_count`, `next_retry_at`.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/db/migrations/003_add_track_analysis.sql \
        backend/internal/db/migrations/schema.sql \
        backend/internal/db/db.go \
        backend/internal/models/models.go \
        backend/tracks/tracksdata.go
git commit -m "feat: add track analysis columns (bpm, key, status) to schema"
```

---

## Task 2: Camelot key mapping (pure function, TDD)

**Files:**
- Create: `backend/analysis/camelot.go`
- Create: `backend/analysis/camelot_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/analysis/camelot_test.go`:

```go
package analysis

import "testing"

func TestToCamelot(t *testing.T) {
    cases := []struct {
        key, scale, want string
    }{
        // Major -> B side
        {"C", "major", "8B"},
        {"G", "major", "9B"},
        {"D", "major", "10B"},
        {"A", "major", "11B"},
        {"E", "major", "12B"},
        {"B", "major", "1B"},
        {"F#", "major", "2B"},
        {"Db", "major", "3B"},
        {"Ab", "major", "4B"},
        {"Eb", "major", "5B"},
        {"Bb", "major", "6B"},
        {"F", "major", "7B"},
        // Minor -> A side
        {"A", "minor", "8A"},
        {"E", "minor", "9A"},
        {"B", "minor", "10A"},
        {"F#", "minor", "11A"},
        {"C#", "minor", "12A"},
        {"G#", "minor", "1A"},
        {"Eb", "minor", "2A"},
        {"Bb", "minor", "3A"},
        {"F", "minor", "4A"},
        {"C", "minor", "5A"},
        {"G", "minor", "6A"},
        {"D", "minor", "7A"},
        // Enharmonic equivalents accepted
        {"Gb", "major", "2B"}, // same as F#
        {"C#", "major", "3B"}, // same as Db
        {"D#", "minor", "2A"}, // same as Eb
    }
    for _, tc := range cases {
        got := ToCamelot(tc.key, tc.scale)
        if got != tc.want {
            t.Errorf("ToCamelot(%q, %q) = %q, want %q", tc.key, tc.scale, got, tc.want)
        }
    }
}

func TestToCamelot_Invalid(t *testing.T) {
    cases := []struct{ key, scale string }{
        {"", "major"},
        {"H", "major"},
        {"C", "phrygian"},
        {"C", ""},
    }
    for _, tc := range cases {
        if got := ToCamelot(tc.key, tc.scale); got != "" {
            t.Errorf("ToCamelot(%q, %q) = %q, want empty", tc.key, tc.scale, got)
        }
    }
}
```

- [ ] **Step 2: Run test, verify fails**

```bash
cd backend && go test ./analysis/...
```

Expected: FAIL with "undefined: ToCamelot".

- [ ] **Step 3: Implement `ToCamelot`**

Create `backend/analysis/camelot.go`:

```go
package analysis

// ToCamelot converts a musical key + scale ("A" + "minor") to Camelot wheel
// notation ("8A"). Returns empty string for unrecognized inputs.
//
// The Camelot wheel indexes the 12 major keys as 1B..12B and the 12 minor
// keys as 1A..12A, ordered so that +1/-1 and same-number-other-letter keys
// are harmonically compatible.
func ToCamelot(key, scale string) string {
    majorToCamelot := map[string]string{
        "C": "8B", "G": "9B", "D": "10B", "A": "11B", "E": "12B", "B": "1B",
        "F#": "2B", "Gb": "2B",
        "Db": "3B", "C#": "3B",
        "Ab": "4B", "G#": "4B",
        "Eb": "5B", "D#": "5B",
        "Bb": "6B", "A#": "6B",
        "F": "7B",
    }
    minorToCamelot := map[string]string{
        "A": "8A", "E": "9A", "B": "10A",
        "F#": "11A", "Gb": "11A",
        "C#": "12A", "Db": "12A",
        "G#": "1A", "Ab": "1A",
        "Eb": "2A", "D#": "2A",
        "Bb": "3A", "A#": "3A",
        "F": "4A", "C": "5A", "G": "6A", "D": "7A",
    }
    switch scale {
    case "major":
        return majorToCamelot[key]
    case "minor":
        return minorToCamelot[key]
    default:
        return ""
    }
}
```

- [ ] **Step 4: Run test, verify passes**

```bash
cd backend && go test ./analysis/... -v
```

Expected: PASS for `TestToCamelot` and `TestToCamelot_Invalid`.

- [ ] **Step 5: Commit**

```bash
git add backend/analysis/camelot.go backend/analysis/camelot_test.go
git commit -m "feat(analysis): Camelot wheel mapping for musical keys"
```

---

## Task 3: Essentia JSON parser (TDD)

**Files:**
- Create: `backend/analysis/essentia_parse.go`
- Create: `backend/analysis/essentia_parse_test.go`
- Create: `backend/analysis/testdata/essentia_success.json`

- [ ] **Step 1: Create a realistic fixture**

Create `backend/analysis/testdata/essentia_success.json` (a trimmed representation of what `streaming_extractor_music` emits):

```json
{
  "rhythm": {
    "bpm": 128.04,
    "bpm_confidence": 3.82
  },
  "tonal": {
    "key_key": "A",
    "key_scale": "minor",
    "key_strength": 0.78
  }
}
```

- [ ] **Step 2: Write the failing test**

Create `backend/analysis/essentia_parse_test.go`:

```go
package analysis

import (
    "os"
    "testing"
)

func TestParseEssentiaOutput_Success(t *testing.T) {
    raw, err := os.ReadFile("testdata/essentia_success.json")
    if err != nil {
        t.Fatalf("read fixture: %v", err)
    }
    got, err := ParseEssentiaOutput(raw)
    if err != nil {
        t.Fatalf("ParseEssentiaOutput: %v", err)
    }
    if got.BPM != 128.04 {
        t.Errorf("BPM = %v, want 128.04", got.BPM)
    }
    // bpm_confidence of 3.82 should normalize to min(3.82/5.0, 1.0) = 0.764
    if got.BPMConfidence < 0.76 || got.BPMConfidence > 0.77 {
        t.Errorf("BPMConfidence = %v, want ~0.764", got.BPMConfidence)
    }
    if got.Key != "8A" {
        t.Errorf("Key = %q, want %q", got.Key, "8A")
    }
    if got.KeyConfidence != 0.78 {
        t.Errorf("KeyConfidence = %v, want 0.78", got.KeyConfidence)
    }
}

func TestParseEssentiaOutput_MalformedJSON(t *testing.T) {
    _, err := ParseEssentiaOutput([]byte("{not json"))
    if err == nil {
        t.Fatal("expected error for malformed JSON")
    }
}

func TestParseEssentiaOutput_MissingBPM(t *testing.T) {
    raw := []byte(`{"tonal":{"key_key":"A","key_scale":"minor","key_strength":0.5}}`)
    _, err := ParseEssentiaOutput(raw)
    if err == nil {
        t.Fatal("expected error when BPM is missing")
    }
}

func TestParseEssentiaOutput_UnknownKey_StillReturnsBPM(t *testing.T) {
    // An invalid scale should leave Key empty but BPM still populated.
    raw := []byte(`{"rhythm":{"bpm":120,"bpm_confidence":2.0},"tonal":{"key_key":"C","key_scale":"phrygian","key_strength":0.5}}`)
    got, err := ParseEssentiaOutput(raw)
    if err != nil {
        t.Fatalf("ParseEssentiaOutput: %v", err)
    }
    if got.BPM != 120 {
        t.Errorf("BPM = %v, want 120", got.BPM)
    }
    if got.Key != "" {
        t.Errorf("Key = %q, want empty (unknown scale)", got.Key)
    }
    if got.KeyConfidence != 0 {
        t.Errorf("KeyConfidence = %v, want 0 when key unknown", got.KeyConfidence)
    }
}

func TestParseEssentiaOutput_ConfidenceClamped(t *testing.T) {
    // bpm_confidence > 5.0 should clamp to 1.0
    raw := []byte(`{"rhythm":{"bpm":120,"bpm_confidence":8.0},"tonal":{"key_key":"A","key_scale":"minor","key_strength":1.5}}`)
    got, err := ParseEssentiaOutput(raw)
    if err != nil {
        t.Fatalf("ParseEssentiaOutput: %v", err)
    }
    if got.BPMConfidence != 1.0 {
        t.Errorf("BPMConfidence = %v, want 1.0", got.BPMConfidence)
    }
    if got.KeyConfidence != 1.0 {
        t.Errorf("KeyConfidence = %v, want 1.0", got.KeyConfidence)
    }
}
```

- [ ] **Step 3: Run test, verify fails**

```bash
cd backend && go test ./analysis/... -run TestParseEssentiaOutput -v
```

Expected: FAIL with "undefined: ParseEssentiaOutput".

- [ ] **Step 4: Implement parser**

Create `backend/analysis/essentia_parse.go`:

```go
package analysis

import (
    "encoding/json"
    "errors"
    "fmt"
)

// Result is the structured output of an essentia analysis run.
type Result struct {
    BPM           float64
    BPMConfidence float64 // normalized to [0, 1]
    Key           string  // Camelot notation, e.g. "8A"; "" if unknown
    KeyConfidence float64 // normalized to [0, 1]; 0 if key unknown
}

type rawEssentia struct {
    Rhythm struct {
        BPM           float64 `json:"bpm"`
        BPMConfidence float64 `json:"bpm_confidence"`
    } `json:"rhythm"`
    Tonal struct {
        KeyKey      string  `json:"key_key"`
        KeyScale    string  `json:"key_scale"`
        KeyStrength float64 `json:"key_strength"`
    } `json:"tonal"`
}

// ParseEssentiaOutput reads the JSON written by streaming_extractor_music and
// returns a normalized Result. Returns an error for malformed JSON or missing
// BPM (the only truly required field).
func ParseEssentiaOutput(raw []byte) (Result, error) {
    var e rawEssentia
    if err := json.Unmarshal(raw, &e); err != nil {
        return Result{}, fmt.Errorf("parse essentia json: %w", err)
    }
    if e.Rhythm.BPM <= 0 {
        return Result{}, errors.New("essentia output missing bpm")
    }
    camelot := ToCamelot(e.Tonal.KeyKey, e.Tonal.KeyScale)
    keyConf := normalizeConfidence(e.Tonal.KeyStrength)
    if camelot == "" {
        keyConf = 0
    }
    return Result{
        BPM:           e.Rhythm.BPM,
        BPMConfidence: normalizeConfidence(e.Rhythm.BPMConfidence / 5.0),
        Key:           camelot,
        KeyConfidence: keyConf,
    }, nil
}

func normalizeConfidence(x float64) float64 {
    if x < 0 {
        return 0
    }
    if x > 1 {
        return 1
    }
    return x
}
```

- [ ] **Step 5: Run test, verify passes**

```bash
cd backend && go test ./analysis/... -run TestParseEssentiaOutput -v
```

Expected: all 5 tests PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/analysis/essentia_parse.go backend/analysis/essentia_parse_test.go \
        backend/analysis/testdata/essentia_success.json
git commit -m "feat(analysis): parse essentia JSON output with confidence normalization"
```

---

## Task 4: Essentia subprocess wrapper (TDD with stub binary)

**Files:**
- Create: `backend/analysis/essentia.go`
- Create: `backend/analysis/essentia_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/analysis/essentia_test.go`:

```go
package analysis

import (
    "context"
    "errors"
    "os"
    "path/filepath"
    "runtime"
    "testing"
    "time"
)

// writeStubBinary creates a fake streaming_extractor_music in a temp dir that
// writes the supplied JSON to its second argument and exits 0. Returns the
// directory path so the caller can prepend it to PATH.
func writeStubBinary(t *testing.T, outputJSON string) string {
    t.Helper()
    if runtime.GOOS == "windows" {
        t.Skip("stub shell script requires a POSIX shell")
    }
    dir := t.TempDir()
    script := "#!/usr/bin/env bash\nprintf '%s' '" + outputJSON + "' > \"$2\"\n"
    path := filepath.Join(dir, "streaming_extractor_music")
    if err := os.WriteFile(path, []byte(script), 0755); err != nil {
        t.Fatalf("write stub: %v", err)
    }
    return dir
}

func withPathPrepended(t *testing.T, dir string) {
    t.Helper()
    prev := os.Getenv("PATH")
    os.Setenv("PATH", dir+string(os.PathListSeparator)+prev)
    t.Cleanup(func() { os.Setenv("PATH", prev) })
}

func TestAnalyzer_Analyze_Success(t *testing.T) {
    stubDir := writeStubBinary(t, `{"rhythm":{"bpm":124.0,"bpm_confidence":4.0},"tonal":{"key_key":"C","key_scale":"major","key_strength":0.9}}`)
    withPathPrepended(t, stubDir)

    audio := filepath.Join(t.TempDir(), "fake.wav")
    os.WriteFile(audio, []byte("fake"), 0644)

    a := NewAnalyzer(30 * time.Second)
    got, err := a.Analyze(context.Background(), audio)
    if err != nil {
        t.Fatalf("Analyze: %v", err)
    }
    if got.BPM != 124.0 {
        t.Errorf("BPM = %v, want 124.0", got.BPM)
    }
    if got.Key != "8B" {
        t.Errorf("Key = %q, want 8B", got.Key)
    }
}

func TestAnalyzer_Analyze_BinaryMissing(t *testing.T) {
    os.Setenv("PATH", "/nonexistent")
    t.Cleanup(func() { os.Setenv("PATH", "/usr/bin:/bin") })

    a := NewAnalyzer(30 * time.Second)
    _, err := a.Analyze(context.Background(), "/tmp/anything.wav")
    if !errors.Is(err, ErrBinaryMissing) {
        t.Fatalf("want ErrBinaryMissing, got %v", err)
    }
}

func TestAnalyzer_Analyze_FileMissing(t *testing.T) {
    // Stub exits 0 but we want the wrapper to return ErrFileMissing before
    // even calling the binary.
    stubDir := writeStubBinary(t, `{"rhythm":{"bpm":120,"bpm_confidence":2.0},"tonal":{"key_key":"A","key_scale":"minor","key_strength":0.5}}`)
    withPathPrepended(t, stubDir)

    a := NewAnalyzer(30 * time.Second)
    _, err := a.Analyze(context.Background(), "/tmp/definitely-does-not-exist-xyz.wav")
    if !errors.Is(err, ErrFileMissing) {
        t.Fatalf("want ErrFileMissing, got %v", err)
    }
}

func TestAnalyzer_Analyze_MalformedOutput(t *testing.T) {
    stubDir := writeStubBinary(t, `{not json`)
    withPathPrepended(t, stubDir)

    audio := filepath.Join(t.TempDir(), "fake.wav")
    os.WriteFile(audio, []byte("fake"), 0644)

    a := NewAnalyzer(30 * time.Second)
    _, err := a.Analyze(context.Background(), audio)
    if !errors.Is(err, ErrMalformedOutput) {
        t.Fatalf("want ErrMalformedOutput, got %v", err)
    }
}
```

- [ ] **Step 2: Run test, verify fails**

```bash
cd backend && go test ./analysis/... -run TestAnalyzer -v
```

Expected: FAIL with "undefined: NewAnalyzer / ErrBinaryMissing / ErrFileMissing / ErrMalformedOutput".

- [ ] **Step 3: Implement the wrapper**

Create `backend/analysis/essentia.go`:

```go
package analysis

import (
    "context"
    "errors"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "time"
)

// Sentinel errors so callers (the manager) can decide retry policy.
var (
    ErrBinaryMissing   = errors.New("essentia binary not found on PATH")
    ErrFileMissing     = errors.New("audio file not found")
    ErrTimeout         = errors.New("essentia analysis timed out")
    ErrMalformedOutput = errors.New("essentia produced malformed output")
)

const binaryName = "streaming_extractor_music"

// Analyzer wraps the streaming_extractor_music binary.
type Analyzer struct {
    timeout time.Duration
}

func NewAnalyzer(timeout time.Duration) *Analyzer {
    return &Analyzer{timeout: timeout}
}

// BinaryAvailable reports whether the essentia binary is on PATH. Call once
// at startup to decide whether to start the ticker.
func BinaryAvailable() bool {
    _, err := exec.LookPath(binaryName)
    return err == nil
}

// Analyze runs the essentia extractor on the given audio file path.
func (a *Analyzer) Analyze(ctx context.Context, audioPath string) (Result, error) {
    if _, err := exec.LookPath(binaryName); err != nil {
        return Result{}, ErrBinaryMissing
    }
    if _, err := os.Stat(audioPath); err != nil {
        if os.IsNotExist(err) {
            return Result{}, ErrFileMissing
        }
        return Result{}, fmt.Errorf("stat audio: %w", err)
    }

    outDir, err := os.MkdirTemp("", "essentia-*")
    if err != nil {
        return Result{}, fmt.Errorf("mktemp: %w", err)
    }
    defer os.RemoveAll(outDir)
    outPath := filepath.Join(outDir, "out.json")

    runCtx, cancel := context.WithTimeout(ctx, a.timeout)
    defer cancel()

    cmd := exec.CommandContext(runCtx, binaryName, audioPath, outPath)
    output, err := cmd.CombinedOutput()
    if runCtx.Err() == context.DeadlineExceeded {
        return Result{}, ErrTimeout
    }
    if err != nil {
        return Result{}, fmt.Errorf("essentia exec failed: %w (output: %s)", err, string(output))
    }

    raw, err := os.ReadFile(outPath)
    if err != nil {
        return Result{}, fmt.Errorf("read essentia output: %w", err)
    }
    result, err := ParseEssentiaOutput(raw)
    if err != nil {
        return Result{}, fmt.Errorf("%w: %v", ErrMalformedOutput, err)
    }
    return result, nil
}
```

- [ ] **Step 4: Run test, verify passes**

```bash
cd backend && go test ./analysis/... -run TestAnalyzer -v
```

Expected: all 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/analysis/essentia.go backend/analysis/essentia_test.go
git commit -m "feat(analysis): essentia subprocess wrapper with sentinel errors"
```

---

## Task 5: Analysis repository (claim/update/mark-failed with in-memory SQLite)

**Files:**
- Create: `backend/analysis/repository.go`
- Create: `backend/analysis/repository_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/analysis/repository_test.go`:

```go
package analysis

import (
    "context"
    "database/sql"
    "testing"
    "time"

    _ "github.com/mattn/go-sqlite3"
)

// newTestDB sets up an in-memory SQLite with just enough schema for the repo.
func newTestDB(t *testing.T) *sql.DB {
    t.Helper()
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatalf("open: %v", err)
    }
    _, err = db.Exec(`
        CREATE TABLE tracks (
            id TEXT PRIMARY KEY,
            owner_user_id TEXT NOT NULL DEFAULT 'u1',
            original_filename TEXT NOT NULL DEFAULT 'f.wav',
            content_type TEXT NOT NULL DEFAULT 'audio/wav',
            size_bytes INTEGER NOT NULL DEFAULT 0,
            file_path TEXT NOT NULL DEFAULT '/x',
            created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
            bpm REAL,
            bpm_confidence REAL,
            musical_key TEXT,
            key_confidence REAL,
            analyzed_at DATETIME,
            analysis_status TEXT NOT NULL DEFAULT 'pending',
            analysis_error TEXT,
            analysis_retry_count INTEGER NOT NULL DEFAULT 0,
            next_retry_at DATETIME
        );
    `)
    if err != nil {
        t.Fatalf("create: %v", err)
    }
    t.Cleanup(func() { db.Close() })
    return db
}

func seedPending(t *testing.T, db *sql.DB, id, filePath string) {
    t.Helper()
    _, err := db.Exec(`INSERT INTO tracks (id, file_path) VALUES (?, ?)`, id, filePath)
    if err != nil {
        t.Fatalf("seed: %v", err)
    }
}

func TestRepository_ClaimNextPending_ReturnsPending(t *testing.T) {
    db := newTestDB(t)
    seedPending(t, db, "t1", "/audio/t1.wav")

    repo := NewRepository(db)
    got, err := repo.ClaimNextPending(context.Background())
    if err != nil {
        t.Fatalf("ClaimNextPending: %v", err)
    }
    if got == nil || got.ID != "t1" {
        t.Fatalf("got %v, want track t1", got)
    }
    if got.FilePath != "/audio/t1.wav" {
        t.Errorf("FilePath = %q", got.FilePath)
    }
}

func TestRepository_ClaimNextPending_NoneAvailable(t *testing.T) {
    db := newTestDB(t)
    repo := NewRepository(db)
    got, err := repo.ClaimNextPending(context.Background())
    if err != nil {
        t.Fatalf("ClaimNextPending: %v", err)
    }
    if got != nil {
        t.Fatalf("expected nil, got %v", got)
    }
}

func TestRepository_ClaimNextPending_SkipsFutureRetry(t *testing.T) {
    db := newTestDB(t)
    seedPending(t, db, "t1", "/a.wav")
    // Push t1's next_retry_at into the future
    _, _ = db.Exec(`UPDATE tracks SET next_retry_at = datetime('now', '+1 hour') WHERE id = 't1'`)

    repo := NewRepository(db)
    got, err := repo.ClaimNextPending(context.Background())
    if err != nil {
        t.Fatalf("ClaimNextPending: %v", err)
    }
    if got != nil {
        t.Fatalf("should skip track with future next_retry_at, got %v", got)
    }
}

func TestRepository_MarkAnalyzed_UpdatesFields(t *testing.T) {
    db := newTestDB(t)
    seedPending(t, db, "t1", "/a.wav")
    repo := NewRepository(db)

    camKey := "8A"
    err := repo.MarkAnalyzed(context.Background(), "t1", Result{
        BPM: 128.0, BPMConfidence: 0.8, Key: camKey, KeyConfidence: 0.78,
    })
    if err != nil {
        t.Fatalf("MarkAnalyzed: %v", err)
    }

    var status string
    var bpm sql.NullFloat64
    var key sql.NullString
    err = db.QueryRow(`SELECT analysis_status, bpm, musical_key FROM tracks WHERE id='t1'`).Scan(&status, &bpm, &key)
    if err != nil {
        t.Fatalf("read: %v", err)
    }
    if status != "analyzed" {
        t.Errorf("status = %q", status)
    }
    if !bpm.Valid || bpm.Float64 != 128.0 {
        t.Errorf("bpm = %v", bpm)
    }
    if !key.Valid || key.String != "8A" {
        t.Errorf("key = %v", key)
    }
}

func TestRepository_RecordFailure_IncrementsAndBackoff(t *testing.T) {
    db := newTestDB(t)
    seedPending(t, db, "t1", "/a.wav")
    repo := NewRepository(db)

    // First failure -> still pending, next_retry_at ~10min out
    err := repo.RecordFailure(context.Background(), "t1", "boom", time.Now())
    if err != nil {
        t.Fatalf("RecordFailure: %v", err)
    }
    var status string
    var retryCount int
    var nextRetry sql.NullTime
    _ = db.QueryRow(`SELECT analysis_status, analysis_retry_count, next_retry_at FROM tracks WHERE id='t1'`).Scan(&status, &retryCount, &nextRetry)
    if status != "pending" {
        t.Errorf("status after 1st failure = %q, want pending", status)
    }
    if retryCount != 1 {
        t.Errorf("retry_count = %d, want 1", retryCount)
    }
    if !nextRetry.Valid {
        t.Fatalf("next_retry_at not set")
    }

    // Third failure -> terminal
    _ = repo.RecordFailure(context.Background(), "t1", "boom", time.Now())
    _ = repo.RecordFailure(context.Background(), "t1", "boom", time.Now())

    _ = db.QueryRow(`SELECT analysis_status, analysis_retry_count FROM tracks WHERE id='t1'`).Scan(&status, &retryCount)
    if status != "failed" {
        t.Errorf("status after 3rd failure = %q, want failed", status)
    }
    if retryCount != 3 {
        t.Errorf("retry_count = %d, want 3", retryCount)
    }
}

func TestRepository_RecordTerminalFailure_SetsFailedImmediately(t *testing.T) {
    db := newTestDB(t)
    seedPending(t, db, "t1", "/a.wav")
    repo := NewRepository(db)

    err := repo.RecordTerminalFailure(context.Background(), "t1", "file missing")
    if err != nil {
        t.Fatalf("RecordTerminalFailure: %v", err)
    }
    var status string
    _ = db.QueryRow(`SELECT analysis_status FROM tracks WHERE id='t1'`).Scan(&status)
    if status != "failed" {
        t.Errorf("status = %q, want failed", status)
    }
}
```

- [ ] **Step 2: Run test, verify fails**

```bash
cd backend && go test ./analysis/... -run TestRepository -v
```

Expected: FAIL with "undefined: NewRepository".

- [ ] **Step 3: Implement the repository**

Create `backend/analysis/repository.go`:

```go
package analysis

import (
    "context"
    "database/sql"
    "time"
)

const maxRetries = 3

// backoffFor returns the time until the next retry attempt given how many
// failures have already been recorded (1-based).
func backoffFor(retryCount int) time.Duration {
    switch retryCount {
    case 1:
        return 10 * time.Minute
    case 2:
        return 1 * time.Hour
    default:
        return 6 * time.Hour
    }
}

// ClaimedTrack holds the minimum info needed to run analysis.
type ClaimedTrack struct {
    ID       string
    FilePath string
}

// Repository reads/writes analysis fields on the tracks table. It accepts a
// *sql.DB directly (not the project's *db.DB wrapper) so tests can use an
// in-memory SQLite.
type Repository struct {
    db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
    return &Repository{db: db}
}

// ClaimNextPending returns the next track eligible for analysis, or nil if
// none. Eligibility: analysis_status='pending' AND next_retry_at is null or
// in the past. Sorted by upload order so backfill drains oldest first.
func (r *Repository) ClaimNextPending(ctx context.Context) (*ClaimedTrack, error) {
    row := r.db.QueryRowContext(ctx, `
        SELECT id, file_path
        FROM tracks
        WHERE analysis_status = 'pending'
          AND (next_retry_at IS NULL OR next_retry_at <= datetime('now'))
        ORDER BY created_at ASC
        LIMIT 1
    `)
    var t ClaimedTrack
    err := row.Scan(&t.ID, &t.FilePath)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    return &t, nil
}

// MarkAnalyzed writes a successful result and flips status to 'analyzed'.
// If the key is empty (unknown), musical_key is set NULL but the row is still
// considered analyzed.
func (r *Repository) MarkAnalyzed(ctx context.Context, id string, res Result) error {
    var key interface{}
    if res.Key != "" {
        key = res.Key
    }
    _, err := r.db.ExecContext(ctx, `
        UPDATE tracks
        SET bpm = ?, bpm_confidence = ?, musical_key = ?, key_confidence = ?,
            analyzed_at = datetime('now'),
            analysis_status = 'analyzed',
            analysis_error = NULL,
            next_retry_at = NULL,
            updated_at = datetime('now')
        WHERE id = ?
    `, res.BPM, res.BPMConfidence, key, res.KeyConfidence, id)
    return err
}

// RecordFailure increments retry_count and schedules the next attempt. After
// maxRetries failures the status flips to 'failed' (terminal).
func (r *Repository) RecordFailure(ctx context.Context, id, errMsg string, now time.Time) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    var current int
    err = tx.QueryRowContext(ctx, `SELECT analysis_retry_count FROM tracks WHERE id=?`, id).Scan(&current)
    if err != nil {
        return err
    }
    next := current + 1
    if next >= maxRetries {
        _, err = tx.ExecContext(ctx, `
            UPDATE tracks
            SET analysis_status = 'failed',
                analysis_retry_count = ?,
                analysis_error = ?,
                next_retry_at = NULL,
                updated_at = datetime('now')
            WHERE id = ?
        `, next, errMsg, id)
    } else {
        retryAt := now.Add(backoffFor(next))
        _, err = tx.ExecContext(ctx, `
            UPDATE tracks
            SET analysis_status = 'pending',
                analysis_retry_count = ?,
                analysis_error = ?,
                next_retry_at = ?,
                updated_at = datetime('now')
            WHERE id = ?
        `, next, errMsg, retryAt, id)
    }
    if err != nil {
        return err
    }
    return tx.Commit()
}

// RecordTerminalFailure flips a track straight to 'failed' with no retries.
// Use for non-recoverable errors (e.g. file missing on disk).
func (r *Repository) RecordTerminalFailure(ctx context.Context, id, errMsg string) error {
    _, err := r.db.ExecContext(ctx, `
        UPDATE tracks
        SET analysis_status = 'failed',
            analysis_error = ?,
            next_retry_at = NULL,
            updated_at = datetime('now')
        WHERE id = ?
    `, errMsg, id)
    return err
}
```

- [ ] **Step 4: Run test, verify passes**

```bash
cd backend && go test ./analysis/... -run TestRepository -v
```

Expected: all 6 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/analysis/repository.go backend/analysis/repository_test.go
git commit -m "feat(analysis): repository for claim/update/fail track analysis"
```

---

## Task 6: Analysis manager (orchestration, TDD)

**Files:**
- Create: `backend/analysis/manager.go`
- Create: `backend/analysis/manager_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/analysis/manager_test.go`:

```go
package analysis

import (
    "context"
    "database/sql"
    "errors"
    "testing"
)

// fakeAnalyzer returns canned results or errors for tests.
type fakeAnalyzer struct {
    result Result
    err    error
    calls  int
}

func (f *fakeAnalyzer) Analyze(ctx context.Context, path string) (Result, error) {
    f.calls++
    return f.result, f.err
}

func TestManager_ProcessOne_NoPending(t *testing.T) {
    db := newTestDB(t)
    repo := NewRepository(db)
    fa := &fakeAnalyzer{}
    m := NewManager(repo, fa)

    processed, err := m.ProcessOne(context.Background())
    if err != nil {
        t.Fatalf("ProcessOne: %v", err)
    }
    if processed {
        t.Error("expected processed=false when nothing to do")
    }
    if fa.calls != 0 {
        t.Errorf("analyzer called %d times, want 0", fa.calls)
    }
}

func TestManager_ProcessOne_Success(t *testing.T) {
    db := newTestDB(t)
    seedPending(t, db, "t1", "/a.wav")
    repo := NewRepository(db)
    fa := &fakeAnalyzer{result: Result{BPM: 128, BPMConfidence: 0.8, Key: "8A", KeyConfidence: 0.7}}
    m := NewManager(repo, fa)

    processed, err := m.ProcessOne(context.Background())
    if err != nil {
        t.Fatalf("ProcessOne: %v", err)
    }
    if !processed {
        t.Error("expected processed=true")
    }

    var status string
    var bpm sql.NullFloat64
    _ = db.QueryRow(`SELECT analysis_status, bpm FROM tracks WHERE id='t1'`).Scan(&status, &bpm)
    if status != "analyzed" || bpm.Float64 != 128 {
        t.Errorf("status=%q bpm=%v", status, bpm)
    }
}

func TestManager_ProcessOne_RetryableFailure(t *testing.T) {
    db := newTestDB(t)
    seedPending(t, db, "t1", "/a.wav")
    repo := NewRepository(db)
    fa := &fakeAnalyzer{err: ErrTimeout}
    m := NewManager(repo, fa)

    _, err := m.ProcessOne(context.Background())
    if err != nil {
        t.Fatalf("ProcessOne: %v", err)
    }

    var status string
    var retryCount int
    _ = db.QueryRow(`SELECT analysis_status, analysis_retry_count FROM tracks WHERE id='t1'`).Scan(&status, &retryCount)
    if status != "pending" {
        t.Errorf("status=%q, want pending after 1st retryable failure", status)
    }
    if retryCount != 1 {
        t.Errorf("retry_count=%d, want 1", retryCount)
    }
}

func TestManager_ProcessOne_TerminalFailureOnMissingFile(t *testing.T) {
    db := newTestDB(t)
    seedPending(t, db, "t1", "/a.wav")
    repo := NewRepository(db)
    fa := &fakeAnalyzer{err: ErrFileMissing}
    m := NewManager(repo, fa)

    _, err := m.ProcessOne(context.Background())
    if err != nil {
        t.Fatalf("ProcessOne: %v", err)
    }

    var status string
    var retryCount int
    _ = db.QueryRow(`SELECT analysis_status, analysis_retry_count FROM tracks WHERE id='t1'`).Scan(&status, &retryCount)
    if status != "failed" {
        t.Errorf("status=%q, want failed (terminal)", status)
    }
    if retryCount != 0 {
        t.Errorf("retry_count=%d, want 0 for terminal failure", retryCount)
    }
}

func TestManager_ProcessOne_BinaryMissingReturnsError(t *testing.T) {
    db := newTestDB(t)
    seedPending(t, db, "t1", "/a.wav")
    repo := NewRepository(db)
    fa := &fakeAnalyzer{err: ErrBinaryMissing}
    m := NewManager(repo, fa)

    // ErrBinaryMissing means config is broken; surface it rather than burning
    // retry budget on every track in the library.
    _, err := m.ProcessOne(context.Background())
    if !errors.Is(err, ErrBinaryMissing) {
        t.Fatalf("want ErrBinaryMissing, got %v", err)
    }

    var status string
    var retryCount int
    _ = db.QueryRow(`SELECT analysis_status, analysis_retry_count FROM tracks WHERE id='t1'`).Scan(&status, &retryCount)
    if status != "pending" || retryCount != 0 {
        t.Errorf("status=%q retry=%d, want untouched", status, retryCount)
    }
}
```

- [ ] **Step 2: Run test, verify fails**

```bash
cd backend && go test ./analysis/... -run TestManager -v
```

Expected: FAIL with "undefined: NewManager".

- [ ] **Step 3: Implement the manager**

Create `backend/analysis/manager.go`:

```go
package analysis

import (
    "context"
    "errors"
    "fmt"
    "time"
)

// analyzer is the narrow interface the manager needs — lets tests swap in a fake.
type analyzer interface {
    Analyze(ctx context.Context, audioPath string) (Result, error)
}

type Manager struct {
    repo     *Repository
    analyzer analyzer
    now      func() time.Time
}

func NewManager(repo *Repository, a analyzer) *Manager {
    return &Manager{repo: repo, analyzer: a, now: time.Now}
}

// ProcessOne claims the next pending track (if any) and analyzes it.
// Returns (processed, err). `processed` is true iff a track was claimed;
// err is only returned for surface-breaking conditions like ErrBinaryMissing.
// Per-track analysis failures are recorded in the DB and do NOT bubble up.
func (m *Manager) ProcessOne(ctx context.Context) (bool, error) {
    claim, err := m.repo.ClaimNextPending(ctx)
    if err != nil {
        return false, fmt.Errorf("claim next pending: %w", err)
    }
    if claim == nil {
        return false, nil
    }

    result, analyzeErr := m.analyzer.Analyze(ctx, claim.FilePath)
    if analyzeErr != nil {
        // Surface ErrBinaryMissing so the ticker can back off instead of
        // marching through every track as failed.
        if errors.Is(analyzeErr, ErrBinaryMissing) {
            return false, analyzeErr
        }
        if errors.Is(analyzeErr, ErrFileMissing) {
            if err := m.repo.RecordTerminalFailure(ctx, claim.ID, analyzeErr.Error()); err != nil {
                fmt.Printf("[analysis] record terminal failure for %s: %v\n", claim.ID, err)
            }
            return true, nil
        }
        // Timeout, malformed output, exec failure — all retryable.
        if err := m.repo.RecordFailure(ctx, claim.ID, analyzeErr.Error(), m.now()); err != nil {
            fmt.Printf("[analysis] record failure for %s: %v\n", claim.ID, err)
        }
        return true, nil
    }

    if err := m.repo.MarkAnalyzed(ctx, claim.ID, result); err != nil {
        fmt.Printf("[analysis] mark analyzed for %s: %v\n", claim.ID, err)
        return true, nil
    }
    return true, nil
}
```

- [ ] **Step 4: Run test, verify passes**

```bash
cd backend && go test ./analysis/... -run TestManager -v
```

Expected: all 5 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/analysis/manager.go backend/analysis/manager_test.go
git commit -m "feat(analysis): manager orchestrates claim, analyze, and status update"
```

---

## Task 7: Ticker loop + server wiring

**Files:**
- Create: `backend/analysis/ticker.go`
- Modify: `backend/main.go`

- [ ] **Step 1: Write the ticker**

Create `backend/analysis/ticker.go`:

```go
package analysis

import (
    "context"
    "errors"
    "fmt"
    "time"
)

// StartLoop polls for pending tracks on the given interval. Stops when ctx
// is cancelled. If Analyze returns ErrBinaryMissing, the loop backs off to
// a longer wait (since no amount of retrying will help until the binary is
// installed and the server restarted).
func StartLoop(ctx context.Context, m *Manager, interval time.Duration) {
    fmt.Printf("[Analysis] Starting loop (interval=%s)\n", interval)
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    backoff := time.Duration(0)
    for {
        select {
        case <-ctx.Done():
            fmt.Println("[Analysis] Loop stopped")
            return
        case <-ticker.C:
            if backoff > 0 {
                // Consume this tick to honor backoff.
                backoff -= interval
                if backoff < 0 {
                    backoff = 0
                }
                continue
            }
            processed, err := m.ProcessOne(ctx)
            if err != nil {
                if errors.Is(err, ErrBinaryMissing) {
                    fmt.Println("[Analysis] essentia binary not available; backing off for 5 minutes")
                    backoff = 5 * time.Minute
                    continue
                }
                fmt.Printf("[Analysis] ProcessOne error: %v\n", err)
            }
            // If processed, immediately loop again to drain the queue faster.
            // Otherwise wait for the next tick.
            if processed {
                // Drain-mode: process next on the following tick, not this one,
                // to keep the Pi responsive.
                _ = processed
            }
        }
    }
}
```

- [ ] **Step 2: Wire into main.go**

In `backend/main.go`, add these imports:

```go
"github.com/faraz525/home-music-server/backend/analysis"
```

Below the soundcloud manager initialization (around line 68), add:

```go
// Initialize analysis (BPM + key detection)
analysisRepo := analysis.NewRepository(db.DB)
analyzer := analysis.NewAnalyzer(90 * time.Second)
analysisManager := analysis.NewManager(analysisRepo, analyzer)
if analysis.BinaryAvailable() {
    fmt.Printf("[CrateDrop] Analysis worker enabled (essentia binary found)\n")
} else {
    fmt.Printf("[CrateDrop] WARNING: streaming_extractor_music not on PATH — analysis disabled\n")
}
```

And below the existing `go soundcloud.StartSyncLoop(...)` line (around line 83), add:

```go
if analysis.BinaryAvailable() {
    go analysis.StartLoop(ctx, analysisManager, 10*time.Second)
}
```

Note: `db.DB` here refers to the embedded `*sql.DB` field on the project's `*db.DB` wrapper (see `backend/internal/db/db.go:16` — `type DB struct{ *sql.DB }`). So the expression `db.DB` unwraps the wrapper.

- [ ] **Step 3: Build**

```bash
cd backend && go build ./...
```

Expected: clean build.

- [ ] **Step 4: Smoke test startup with no binary**

```bash
PATH=/usr/bin:/bin DATA_DIR=/tmp/cratedrop-test-data2 PORT=8082 go run . &
SERVER_PID=$!
sleep 2
kill $SERVER_PID
```

Expected: server log shows the "WARNING: streaming_extractor_music not on PATH" line and exits cleanly on SIGTERM.

- [ ] **Step 5: Commit**

```bash
git add backend/analysis/ticker.go backend/main.go
git commit -m "feat(analysis): background ticker loop and server wiring"
```

---

## Task 8: PATCH /api/tracks/:id for user overrides (TDD)

**Files:**
- Create: `backend/tracks/patch_handler.go`
- Create: `backend/tracks/patch_handler_test.go`
- Modify: `backend/tracks/tracksmanager.go` — add `UpdateAnalysis` method
- Modify: `backend/tracks/tracksdata.go` — add `UpdateAnalysis` repo method
- Modify: `backend/tracks/routes.go` — register PATCH route

- [ ] **Step 1: Add the repo method**

In `backend/tracks/tracksdata.go`, append:

```go
// UpdateAnalysisOverride sets user-edited BPM and/or key and flips status to 'user_edited'.
// nil values leave the corresponding column untouched.
func (r *Repository) UpdateAnalysisOverride(ctx context.Context, trackID string, bpm *float64, musicalKey *string) error {
    // Build a dynamic SET clause so we only update provided fields.
    sets := []string{"analysis_status = 'user_edited'", "analysis_error = NULL", "next_retry_at = NULL", "updated_at = CURRENT_TIMESTAMP"}
    args := []any{}
    if bpm != nil {
        sets = append(sets, "bpm = ?")
        args = append(args, *bpm)
    }
    if musicalKey != nil {
        sets = append(sets, "musical_key = ?")
        args = append(args, *musicalKey)
    }
    args = append(args, trackID)
    query := "UPDATE tracks SET " + strings.Join(sets, ", ") + " WHERE id = ?"
    _, err := r.db.ExecContext(ctx, query, args...)
    return err
}
```

Ensure `import "strings"` is present.

- [ ] **Step 2: Add the manager method**

In `backend/tracks/tracksmanager.go`, append:

```go
// UpdateAnalysisOverride applies a user override to BPM and/or musical key.
// Caller has already validated ranges and authorized the request.
func (m *Manager) UpdateAnalysisOverride(ctx context.Context, trackID string, bpm *float64, musicalKey *string) error {
    return m.repo.UpdateAnalysisOverride(ctx, trackID, bpm, musicalKey)
}
```

- [ ] **Step 3: Write the failing handler test**

Create `backend/tracks/patch_handler_test.go`:

```go
package tracks

import (
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/gin-gonic/gin"
)

func TestValidateKey(t *testing.T) {
    valid := []string{"1A", "12B", "7A", "9B"}
    for _, k := range valid {
        if !isValidCamelot(k) {
            t.Errorf("isValidCamelot(%q) = false, want true", k)
        }
    }
    invalid := []string{"", "0A", "13A", "1C", "1a", "A1", "100A"}
    for _, k := range invalid {
        if isValidCamelot(k) {
            t.Errorf("isValidCamelot(%q) = true, want false", k)
        }
    }
}

func TestValidateBPM(t *testing.T) {
    if !isValidBPM(120) {
        t.Error("isValidBPM(120) = false, want true")
    }
    if isValidBPM(49) {
        t.Error("isValidBPM(49) = true, want false")
    }
    if isValidBPM(251) {
        t.Error("isValidBPM(251) = true, want false")
    }
}

// TestPatchHandler_RejectsInvalidBody verifies the handler rejects malformed
// payloads with a 400 response. Uses a stub manager via an interface-compatible
// no-op so we don't need a full DB.
func TestPatchHandler_RejectsInvalidBody(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := gin.New()
    r.PATCH("/api/tracks/:id", PatchHandler(nil))

    req := httptest.NewRequest("PATCH", "/api/tracks/abc", strings.NewReader(`{"bpm": 9999}`))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != 400 {
        t.Errorf("status = %d, want 400", w.Code)
    }
}

func TestPatchHandler_RejectsEmptyBody(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := gin.New()
    r.PATCH("/api/tracks/:id", PatchHandler(nil))

    req := httptest.NewRequest("PATCH", "/api/tracks/abc", strings.NewReader(`{}`))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != 400 {
        t.Errorf("status = %d, want 400 (empty body)", w.Code)
    }
}
```

- [ ] **Step 4: Run test, verify fails**

```bash
cd backend && go test ./tracks/... -run TestPatch -v
cd backend && go test ./tracks/... -run TestValidate -v
```

Expected: FAIL with "undefined: PatchHandler / isValidCamelot / isValidBPM".

- [ ] **Step 5: Implement the handler**

Create `backend/tracks/patch_handler.go`:

```go
package tracks

import (
    "context"
    "net/http"
    "regexp"

    "github.com/gin-gonic/gin"
)

type patchTrackRequest struct {
    BPM        *float64 `json:"bpm,omitempty"`
    MusicalKey *string  `json:"musical_key,omitempty"`
}

var camelotRE = regexp.MustCompile(`^(1[0-2]|[1-9])[AB]$`)

func isValidBPM(bpm float64) bool {
    return bpm >= 50 && bpm <= 250
}

func isValidCamelot(k string) bool {
    return camelotRE.MatchString(k)
}

// PatchHandler handles PATCH /api/tracks/:id for user overrides of BPM/key.
// `mgr` can be nil only in tests that exercise validation paths.
func PatchHandler(mgr *Manager) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req patchTrackRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_body", "message": err.Error()}})
            return
        }
        if req.BPM == nil && req.MusicalKey == nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "empty_patch", "message": "at least one of bpm, musical_key is required"}})
            return
        }
        if req.BPM != nil && !isValidBPM(*req.BPM) {
            c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_bpm", "message": "bpm must be between 50 and 250"}})
            return
        }
        if req.MusicalKey != nil && !isValidCamelot(*req.MusicalKey) {
            c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_key", "message": "musical_key must be Camelot notation like 8A or 12B"}})
            return
        }

        trackID := c.Param("id")
        // Guard against nil in tests.
        if mgr == nil {
            c.Status(http.StatusNoContent)
            return
        }

        userID, _ := c.Get("user_id")
        track, err := mgr.GetTrack(c.Request.Context(), trackID)
        if err != nil {
            c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "not_found", "message": "track not found"}})
            return
        }
        if track.OwnerUserID != userID {
            // Admin middleware is a separate route; owners-only for PATCH.
            c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "forbidden", "message": "not your track"}})
            return
        }

        if err := mgr.UpdateAnalysisOverride(context.Background(), trackID, req.BPM, req.MusicalKey); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "update_failed", "message": err.Error()}})
            return
        }
        updated, err := mgr.GetTrack(c.Request.Context(), trackID)
        if err != nil {
            c.Status(http.StatusNoContent)
            return
        }
        c.JSON(http.StatusOK, gin.H{"success": true, "data": updated})
    }
}
```

- [ ] **Step 6: Register the route**

In `backend/tracks/routes.go`, inside `Routes(...)`, right after `g.GET("/:id", GetHandler(m))`, add:

```go
g.PATCH("/:id", PatchHandler(m))
```

- [ ] **Step 7: Run tests, verify passes**

```bash
cd backend && go test ./tracks/... -run "TestPatch|TestValidate" -v
```

Expected: all 4 tests PASS.

- [ ] **Step 8: Build**

```bash
cd backend && go build ./...
```

Expected: clean.

- [ ] **Step 9: Commit**

```bash
git add backend/tracks/patch_handler.go backend/tracks/patch_handler_test.go \
        backend/tracks/tracksmanager.go backend/tracks/tracksdata.go \
        backend/tracks/routes.go
git commit -m "feat(tracks): PATCH /api/tracks/:id for user BPM/key overrides"
```

---

## Task 9: Frontend Track type and API client

**Files:**
- Modify: `frontend/src/types/crates.ts`
- Modify: `frontend/src/lib/api.ts`
- Modify: `frontend/src/hooks/useQueries.ts`

- [ ] **Step 1: Extend the Track type**

In `frontend/src/types/crates.ts`, locate the `Track` interface (or export type) and add the optional analysis fields. If it does not exist yet, find the `TrackList` type; the `Track` shape it contains must grow these fields:

```ts
export interface Track {
  id: string
  owner_user_id: string
  original_filename: string
  content_type: string
  size_bytes: number
  duration_seconds?: number
  title?: string
  artist?: string
  album?: string
  genre?: string
  year?: number
  sample_rate?: number
  bitrate?: number
  bpm?: number
  bpm_confidence?: number
  musical_key?: string
  key_confidence?: number
  analyzed_at?: string
  analysis_status: 'pending' | 'analyzed' | 'failed' | 'user_edited'
  file_path: string
  created_at: string
  updated_at: string
}
```

If the file currently imports `Track` from elsewhere, update that definition instead. If no explicit `Track` interface exists (inferred from API responses), add one in this file and export it.

- [ ] **Step 2: Add the patch API method**

In `frontend/src/lib/api.ts`, extend the `tracksApi` object:

```ts
export const tracksApi = {
  getUnsorted: (params?: UnsortedParams) =>
    api.get('/api/tracks', { params: { ...(params || {}), playlist_id: 'unsorted' } }),

  download: (id: string, filename: string) => {
    const link = document.createElement('a')
    link.href = `/api/tracks/${id}/download`
    link.download = filename
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  },

  patch: (id: string, payload: { bpm?: number; musical_key?: string }) =>
    api.patch(`/api/tracks/${id}`, payload),
}
```

- [ ] **Step 3: Add the mutation hook**

In `frontend/src/hooks/useQueries.ts`, append:

```ts
export function useUpdateTrackAnalysis() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: { bpm?: number; musical_key?: string } }) =>
      tracksApi.patch(id, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tracks'] })
    },
  })
}
```

- [ ] **Step 4: Typecheck**

From the `frontend` directory:

```bash
cd frontend && npx tsc --noEmit
```

Expected: zero errors.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/types/crates.ts frontend/src/lib/api.ts frontend/src/hooks/useQueries.ts
git commit -m "feat(frontend): Track type + patch API + useUpdateTrackAnalysis mutation"
```

---

## Task 10: Frontend track list — BPM and Key columns with confidence dot

**Files:**
- Modify: `frontend/src/pages/LibraryPage.tsx`

This task only adds display; inline editing comes in Task 11.

- [ ] **Step 1: Add a ConfidenceDot component at the top of the file**

After the `MiniVinyl` component (around line 27), add:

```tsx
function ConfidenceDot({ status, confidence }: { status: string; confidence?: number }) {
  let color = 'bg-transparent border border-crate-subtle' // pending
  let title = 'Analyzing…'

  if (status === 'failed') {
    color = 'bg-transparent border border-crate-danger'
    title = 'Analysis failed'
  } else if (status === 'user_edited') {
    return null // no dot for user-edited values
  } else if (status === 'analyzed' && typeof confidence === 'number') {
    if (confidence > 0.7) {
      color = 'bg-green-500'
      title = `High confidence (${Math.round(confidence * 100)}%)`
    } else if (confidence >= 0.4) {
      color = 'bg-amber-500'
      title = `Medium confidence (${Math.round(confidence * 100)}%)`
    } else {
      color = 'bg-red-500'
      title = `Low confidence (${Math.round(confidence * 100)}%)`
    }
  }

  return <span className={`inline-block w-2 h-2 rounded-full ${color}`} title={title} />
}
```

- [ ] **Step 2: Update the header grid to include BPM and Key columns**

Find the header row (around line 316) and change `grid-cols-[28px_1fr_1fr_100px_60px_40px]` to `grid-cols-[28px_1fr_1fr_70px_70px_100px_60px_40px]`. Insert the two header cells after `<div>Album</div>`:

```tsx
<div className="text-right">BPM</div>
<div className="text-right">Key</div>
```

- [ ] **Step 3: Update the row grid to match**

Find the track row grid (around line 369) and apply the same column template on the `sm:grid-cols-*` part: change to `sm:grid-cols-[28px_1fr_1fr_70px_70px_100px_60px_40px]`.

Insert two new display cells right after the Album div (around line 451):

```tsx
{/* Desktop: BPM */}
<div className="hidden sm:flex items-center justify-end gap-1.5 text-sm text-crate-subtle tabular-nums">
  <ConfidenceDot status={t.analysis_status} confidence={t.bpm_confidence} />
  <span>{t.bpm ? t.bpm.toFixed(1) : '—'}</span>
</div>

{/* Desktop: Key */}
<div className="hidden sm:flex items-center justify-end gap-1.5 text-sm text-crate-subtle">
  <ConfidenceDot status={t.analysis_status} confidence={t.key_confidence} />
  <span>{t.musical_key || '—'}</span>
</div>
```

- [ ] **Step 4: Build and visually verify**

Run the dev server:

```bash
cd frontend && npm run dev
```

Open the library page in a browser. With no tracks analyzed yet, both columns should render "—" with a grey-outlined dot (pending status).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/pages/LibraryPage.tsx
git commit -m "feat(frontend): BPM and Key columns with confidence indicator"
```

---

## Task 11: Frontend inline edit for BPM and Key

**Files:**
- Modify: `frontend/src/pages/LibraryPage.tsx`

- [ ] **Step 1: Add an EditableCell component**

At the top of the file (after `ConfidenceDot`), add:

```tsx
import type { ReactNode } from 'react'

function EditableCell({
  value,
  onSave,
  validate,
  display,
  align = 'right',
}: {
  value: string
  onSave: (next: string) => Promise<void> | void
  validate: (next: string) => boolean
  display: ReactNode
  align?: 'left' | 'right'
}) {
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(value)
  const [saving, setSaving] = useState(false)

  if (!editing) {
    return (
      <div
        onDoubleClick={(e) => { e.stopPropagation(); setDraft(value); setEditing(true) }}
        className="cursor-text select-none"
        title="Double-click to edit"
      >
        {display}
      </div>
    )
  }

  const commit = async () => {
    if (!validate(draft)) return
    setSaving(true)
    try {
      await onSave(draft)
    } finally {
      setSaving(false)
      setEditing(false)
    }
  }

  return (
    <input
      autoFocus
      value={draft}
      disabled={saving}
      onChange={(e) => setDraft(e.target.value)}
      onBlur={commit}
      onClick={(e) => e.stopPropagation()}
      onKeyDown={(e) => {
        if (e.key === 'Enter') commit()
        else if (e.key === 'Escape') setEditing(false)
      }}
      className={`input h-6 px-1 py-0 text-sm w-16 ${align === 'right' ? 'text-right' : ''}`}
    />
  )
}
```

- [ ] **Step 2: Wire up the mutation and swap display cells for EditableCell**

At the top of `LibraryPage`, add to the existing hooks:

```tsx
import { useCrates, useTracks, useDeleteTrack, useAddTracksToCrate, useRemoveTracksFromCrate, useUpdateTrackAnalysis } from '../hooks/useQueries'
```

Inside the component (near other mutations):

```tsx
const updateAnalysisMutation = useUpdateTrackAnalysis()

const saveBpm = useCallback(async (trackId: string, raw: string) => {
  const bpm = parseFloat(raw)
  if (!isFinite(bpm) || bpm < 50 || bpm > 250) {
    toast.error('BPM must be between 50 and 250')
    return
  }
  try {
    await updateAnalysisMutation.mutateAsync({ id: trackId, payload: { bpm } })
    toast.success('BPM updated')
  } catch {
    toast.error('Failed to update BPM')
  }
}, [updateAnalysisMutation, toast])

const saveKey = useCallback(async (trackId: string, raw: string) => {
  const normalized = raw.trim().toUpperCase()
  if (!/^(1[0-2]|[1-9])[AB]$/.test(normalized)) {
    toast.error('Key must be Camelot notation (e.g. 8A, 12B)')
    return
  }
  try {
    await updateAnalysisMutation.mutateAsync({ id: trackId, payload: { musical_key: normalized } })
    toast.success('Key updated')
  } catch {
    toast.error('Failed to update key')
  }
}, [updateAnalysisMutation, toast])
```

Replace the plain BPM and Key cells added in Task 10 with EditableCell wrappers:

```tsx
{/* Desktop: BPM (editable) */}
<div className="hidden sm:flex items-center justify-end gap-1.5 text-sm text-crate-subtle tabular-nums">
  <ConfidenceDot status={t.analysis_status} confidence={t.bpm_confidence} />
  <EditableCell
    value={t.bpm ? t.bpm.toFixed(1) : ''}
    display={<span>{t.bpm ? t.bpm.toFixed(1) : '—'}</span>}
    validate={(v) => { const n = parseFloat(v); return isFinite(n) && n >= 50 && n <= 250 }}
    onSave={(v) => saveBpm(t.id, v)}
  />
</div>

{/* Desktop: Key (editable) */}
<div className="hidden sm:flex items-center justify-end gap-1.5 text-sm text-crate-subtle">
  <ConfidenceDot status={t.analysis_status} confidence={t.key_confidence} />
  <EditableCell
    value={t.musical_key || ''}
    display={<span>{t.musical_key || '—'}</span>}
    validate={(v) => /^(1[0-2]|[1-9])[AB]$/i.test(v.trim())}
    onSave={(v) => saveKey(t.id, v)}
  />
</div>
```

- [ ] **Step 3: Typecheck**

```bash
cd frontend && npx tsc --noEmit
```

Expected: zero errors.

- [ ] **Step 4: Manual verification**

With the dev server running, double-click a BPM cell, enter `128.0`, press Enter. The value should persist across a page reload and the confidence dot should disappear (user_edited status).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/pages/LibraryPage.tsx
git commit -m "feat(frontend): inline edit for BPM and Key on track list"
```

---

## Task 12: End-to-end smoke test with a real audio file

**Files:** no new files, manual verification

- [ ] **Step 1: Confirm essentia is installed on the dev machine**

```bash
which streaming_extractor_music
```

If missing on macOS: `brew install essentia --HEAD`. If missing on Debian/Pi: `sudo apt install essentia-examples` (availability varies by distro — for Pi OS Bookworm it should work; otherwise build from source via `https://essentia.upf.edu/`).

- [ ] **Step 2: Start a clean dev server**

```bash
rm -rf /tmp/cratedrop-e2e
DATA_DIR=/tmp/cratedrop-e2e PORT=8083 go run ./backend &
SERVER_PID=$!
sleep 2
```

Expected: log includes "Analysis worker enabled (essentia binary found)" and "Starting loop (interval=10s)".

- [ ] **Step 3: Create a user and upload a known track**

Use the frontend at `http://localhost:8083` (or the dev server proxy). Sign up, upload a short WAV file you know the BPM/key of (e.g. a 120 BPM house loop).

- [ ] **Step 4: Wait for analysis**

Watch the server log. Within ~60 seconds you should see no errors and the track's row in SQLite should have `analysis_status='analyzed'`:

```bash
sqlite3 /tmp/cratedrop-e2e/db/cratedrop.sqlite \
  "SELECT id, original_filename, bpm, musical_key, analysis_status FROM tracks;"
```

Expected: row shows a BPM value ±2 of the true tempo and a plausible Camelot key.

- [ ] **Step 5: Verify UI**

Refresh the library page. BPM and Key columns should show the detected values with confidence dots.

- [ ] **Step 6: Verify inline edit**

Double-click the BPM cell, change to a wrong-on-purpose value, save. Refresh. Value persists and confidence dot is gone.

- [ ] **Step 7: Clean up**

```bash
kill $SERVER_PID
```

- [ ] **Step 8: Commit (if any doc touchups happened)**

No code changes expected. If the task surfaced bugs, fix them in a follow-up commit before moving on.

---

## Task 13: Deploy-prereq checks and README note

**Files:**
- Modify: `deploy-local.sh`
- Modify: `deploy-prod.sh`
- Modify: `README.md`

- [ ] **Step 1: Add a binary check to each deploy script**

At the top of `deploy-local.sh` and `deploy-prod.sh` (after the shebang), add:

```bash
if ! command -v streaming_extractor_music >/dev/null 2>&1; then
  echo "WARNING: streaming_extractor_music not found on PATH."
  echo "         BPM/key analysis will be disabled until it is installed."
  echo "         Debian/Pi OS: sudo apt install essentia-examples"
  echo "         macOS:        brew install essentia --HEAD"
fi
```

Do NOT make this a hard failure — the server must still start.

- [ ] **Step 2: Document the prereq in the README**

Add a short section (under existing "Deployment" or similar):

```markdown
## Optional: BPM & Musical Key Analysis

CrateDrop can auto-detect BPM and musical key (Camelot notation) for uploaded
tracks. This requires the `streaming_extractor_music` binary from essentia to
be on PATH.

- Debian / Raspberry Pi OS: `sudo apt install essentia-examples`
- macOS: `brew install essentia --HEAD`

If the binary is not available, the server logs a warning at startup and
disables the analysis worker — upload and playback continue to work normally.
Tracks remain in `pending` status and are analyzed automatically the first
time the server starts with the binary installed.
```

- [ ] **Step 3: Commit**

```bash
git add deploy-local.sh deploy-prod.sh README.md
git commit -m "docs: deploy prereqs and README note for essentia analysis"
```

---

## Done

At this point:

- Every existing track and every future upload is auto-analyzed for BPM and key.
- The library page shows BPM and Key columns with confidence dots.
- Users can double-click to override wrong detections; overrides persist.
- The analysis worker gracefully disables itself if essentia is missing.
- All new logic has unit tests; aggregate backend coverage for the `analysis/` package meets the 80% bar.

Final verification:

```bash
cd backend && go test ./... -cover
cd frontend && npx tsc --noEmit
```

Expected: all tests pass; `backend/analysis` coverage ≥ 80%.
