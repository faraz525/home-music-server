## CrateDrop v0 — Requirements (Raspberry Pi friendly)

### Overview
CrateDrop is a minimalist web app for DJs to securely upload, organize, and stream their own music library. It targets a Raspberry Pi deployment with an attached SSD for storage, acting like a private “Dropbox for tracks” with a sleek web player.

### Goals
- **Invite‑only access**: Only users with an invite can sign up; login/logout/refresh flows.
- **Upload**: Accept WAV/AIFF/FLAC/MP3 via web UI.
- **Library**: View, search, and manage uploaded tracks.
- **Playback**: Stream with seek support (HTTP Range); simple, modern player UI.
- **Admin**: Generate and manage invite codes; view users.

### Non‑Goals (v0)
- Public sharing/links, crates/playlists, comments.
- Collaborative editing, social features.
- Advanced audio processing (waveforms, BPM/key detection, transcoded previews).
- Mobile apps (web works on mobile).

## Personas & Use Cases
- **Admin DJ (owner)**: Sets up the Pi, invites friends, manages storage.
- **Invited DJ (user)**: Uploads personal tracks, streams them, deletes as needed.

- **Use cases**:
  - Sign up using an invite code, login, and stay logged in with refresh tokens.
  - Drag‑and‑drop upload a mix of WAV/FLAC/MP3 files to SSD storage.
  - Browse/search tracks by filename/title/artist.
  - Play tracks in the browser with seek and playback controls.
  - Admin generates invite codes and revokes unused ones.

## Functional Requirements
### Authentication & Authorization
- Users can sign up only with a valid invite code.
- Email + password authentication; passwords are hashed (bcrypt or argon2).
- JWT access tokens (short‑lived) and rotating refresh tokens (httpOnly cookie).
- Endpoints to login, refresh, logout, fetch current user.
- Roles: `admin`, `user`. Admin‑only invite management.

### Invites
- Admin can create single‑use invite codes with optional expiration.
- Viewing list of invites and their status (unused/used/expired).
- Using an invite marks it as used and associates with the signing‑up user.

### Track Upload & Management
- Accept file types: WAV, AIFF, FLAC, MP3.
- Max upload size configurable (default 2GB).
- Server stores uploaded file to SSD under a deterministic path per user and track.
- Extract metadata with `ffprobe` (duration; tags if available).
- List tracks with paging and simple search (by filename/title/artist).
- Users can delete their own tracks; admin can delete any track.

### Streaming & Playback
- Streaming endpoint supports HTTP Range requests for seeking.
- Browser playback should work for MP3 and (where supported) FLAC/WAV.
- Frontend includes a sticky bottom mini‑player with play/pause/seek and next/prev.

### Admin Capabilities
- Create/list invite codes.
- View users (email, created_at, role).

## Non‑Functional Requirements
### Performance
- Handle individual uploads up to 2GB without exhausting memory (stream to disk).
- Concurrent uploads/streams (at least 3 concurrent users on Pi 4/5).

### Security
- Hash passwords (bcrypt/argon2). Never store plaintext.
- Access token lifetime 5–15 minutes; refresh tokens rotate and are stored server‑side (hashed) or invalidated on logout.
- Validate content type and limit size; sanitize filenames; store with server‑generated IDs.
- CORS allowlist for dev; production served from same origin via reverse proxy.

### Reliability & Operations
- SSD mounted at `/mnt/music` with correct ownership/permissions.
- Uploads are atomic: temp file → move to final location after successful write.
- Server restart leaves no orphan temp files.
- Health endpoint for monitoring.

### Maintainability
- Clear module boundaries (auth, invites, uploads, streaming).
- Linting/formatting enforced.

## System Architecture
### Components
- **Frontend**: React + Vite + TypeScript + Tailwind. Built assets served by Nginx.
- **Backend**: Go (Gin or Fiber). JWT auth, file streaming, ffprobe integration.
- **Database**: Primary Postgres 16 (arm64) via Docker, or SQLite single‑file (alternative for simpler ops). Choose one for v0; both are supported via an abstraction.
- **Storage**: Local filesystem on SSD mounted at `/mnt/music`.
- **Reverse proxy**: Nginx serves SPA and proxies `/api` to backend.

### Deployment Target
- Raspberry Pi (arm64). Docker Compose orchestrates Nginx + Backend (+ Postgres if used).
- SSD bind‑mounted into containers for read/write storage.

## Data Model (SQL‑backed)
- `users`: id (uuid), email (unique), password_hash, role, created_at.
- `invites`: code (unique), created_by (user id), used_by (nullable), expires_at, created_at.
- `tracks`: id (uuid), owner_user_id, original_filename, content_type, size_bytes, duration_seconds, title, artist, album, path, created_at.
- `refresh_tokens` (or sessions): id, user_id, token_hash, expires_at, created_at, revoked_at (nullable).

Notes:
- `path` is a server‑managed absolute path under `/mnt/music/cratedrop/...`.
- For SQLite, use appropriate pragmas and WAL mode; for Postgres, enforce constraints and indexes.

## API Specification (v0)
Base path: `/api`

### Auth
- POST `/auth/signup?invite=CODE` — body: { email, password } — returns user + tokens
- POST `/auth/login` — body: { email, password } — returns user + tokens
- POST `/auth/refresh` — rotates tokens; refresh token via httpOnly cookie
- POST `/auth/logout` — invalidates refresh token
- GET `/me` — returns current user profile

### Invites (admin)
- POST `/invites` — body: { expires_at? } — returns { code }
- GET `/invites` — list invites with status

### Tracks
- POST `/tracks` — multipart form: `file`, optional `title`, `artist`, `album`
- GET `/tracks` — query: `q`, `limit`, `cursor` — lists tracks
- GET `/tracks/:id` — metadata for a single track
- GET `/tracks/:id/stream` — audio bytes with Range support
- DELETE `/tracks/:id` — delete (owner or admin)

### Errors
- JSON shape: { error: { code, message } }
- Use appropriate HTTP status codes; include request id in logs.

## Storage Layout (SSD)
Root: `/mnt/music/cratedrop/`
- `library/<user_id>/<track_id>/original.<ext>`
- `previews/<track_id>/preview-128k.aac` (future)
- `db/` (optional for SQLite) and `backups/` (DB backup copies)

Permissions:
- Ensure the container user (backend) can read/write; match host uid/gid or use `:rw` bind mount.

## Frontend Requirements
- **Pages**: Login, Signup (invite), Library (list/search/upload), Admin (invites).
- **Player**: Sticky bottom mini‑player with play/pause/seek, next/prev.
- **Upload UX**: Drag‑and‑drop, progress bar, optimistic row, toast notifications.
- **Auth UX**: Persist session with refresh token; minimal PII stored client‑side.
- **Styling**: Tailwind; light/dark ready but optional for v0.

## Configuration & Environment
- `.env` variables (backend):
  - `APP_ENV=development|production`
  - `JWT_SECRET`, `REFRESH_SECRET`
  - `BASE_URL=http(s)://<host>`
  - `DATA_DIR=/mnt/music/cratedrop`
  - `DATABASE_URL=postgres://...` (if Postgres) or `SQLITE_PATH=...` (if SQLite)
- Nginx:
  - `client_max_body_size 2048M;`
  - Proxy read/send timeouts for long uploads.

## Operations
- Health: `GET /api/healthz` returns 200 + version.
- Logging: structured logs with request id and user id.
- Backups: nightly DB backup copy to `/mnt/music/cratedrop/backups/`.
- Optional remote access: Tailscale or Cloudflare Tunnel for HTTPS without port forwards.

## Dependencies
- Raspberry Pi OS (arm64), Docker + Docker Compose.
- `ffmpeg` (`ffprobe`) installed in backend container (or host) for metadata.
- Go toolchain for development; Node.js for frontend dev.

## Risks & Constraints
- Browser support for FLAC/WAV can vary; MP3 broadly supported.
- Large uploads on weak connections may timeout; proxy/server timeouts must be tuned.
- SSD permissions must be correct or uploads will fail.

## Milestones & Acceptance Criteria
### M0 – Skeleton
- Nginx serves SPA; `/api/healthz` returns OK via proxy.

### M1 – Auth + Invites
- Signup with invite, login, refresh, logout; admin can create invites.
- Tokens rotate; basic rate limiting in place.

### M2 – Upload + Library
- Upload to SSD, metadata extraction, list/search, delete; restart‑safe temp handling.
- 1GB+ file upload verified.

### M3 – Streaming + Player
- Range requests supported; playback works in Chrome/Safari/Firefox.
- Sticky player with seek and next/prev.

### M4 – Polish + Ops
- Error/empty states; request/user logging; DB backup script.
- No console errors in UI.

## Definition of Done (v0)
- End‑to‑end flow: invite → signup → login → upload → stream → delete.
- Deployed on Raspberry Pi with SSD at `/mnt/music`, durable across restarts.
- Basic monitoring via health endpoint; backups configured.

## Appendices
### Suggested Directory & Files
- Backend service under `backend/` (Go), frontend under `frontend/` (React).
- Reverse proxy config in `frontend/ngnix.conf` (or `nginx.conf`), mounted by Compose.
- Compose file `docker-compose.yml` manages Nginx, backend, and optionally Postgres.


