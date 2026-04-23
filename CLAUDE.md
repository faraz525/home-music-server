# CrateDrop - Home Music Server

This is a personal music server for uploading, organizing, and streaming music from a Raspberry Pi.

## Tech Stack

- **Backend**: Go 1.23 + Gin framework
- **Frontend**: React 18 + TypeScript + Vite + Tailwind CSS
- **Database**: SQLite3
- **Containerization**: Docker + docker-compose

## Project Structure

```
home-music-server/
├── backend/
│   ├── auth/           # JWT authentication
│   ├── internal/
│   │   ├── db/         # Database connection and migrations
│   │   ├── handlers/   # HTTP handlers
│   │   └── models/     # Data models
│   ├── playlists/      # Playlist management
│   ├── soundcloud/     # SoundCloud sync feature (yt-dlp based)
│   ├── tracks/         # Track upload/streaming
│   ├── server/         # Server setup
│   ├── main.go         # Entry point
│   └── Dockerfile
├── frontend/
│   ├── src/
│   │   ├── components/ # React components
│   │   ├── pages/      # Page components
│   │   ├── lib/        # API client, utilities
│   │   └── App.tsx     # Router setup
│   └── Dockerfile
└── docker-compose.yml
```

## Deployment (Raspberry Pi)

**Important**: BuildKit causes npm to hang on Raspberry Pi. Always use legacy builder:

```bash
# Rebuild and deploy
cd ~/Documents/home-music-server
docker compose down
DOCKER_BUILDKIT=0 docker compose build --no-cache
docker compose up -d

# View logs
docker logs -f cratedrop-backend
docker logs -f cratedrop-frontend
```

### Data Locations

| Data | Path |
|------|------|
| SQLite DB | `/mnt/ssd/apps/cratedrop/db/cratedrop.sqlite` |
| Music files | `/mnt/ssd/apps/cratedrop/library/` |
| Application logs | `./logs/` |

### Ports

- Backend API: `8080`
- Frontend (nginx): `80`
- Public URL: `cratedrop.farazws.com` (via Cloudflare Tunnel)

## Development

### Backend

```bash
cd backend
go run main.go
```

Environment variables:
- `APP_ENV` - production/development
- `GIN_MODE` - release/debug
- `SQLITE_PATH` - Database file path
- `DATA_DIR` - Base data directory
- `JWT_SECRET` - JWT signing secret
- `REFRESH_SECRET` - Refresh token secret

### Frontend

```bash
cd frontend
npm install
npm run dev
```

The dev server proxies `/api` requests to `localhost:8080`.

## Database

Migrations are in `backend/internal/db/migrations/`. They run automatically on startup.

```bash
# Direct database access
sqlite3 /mnt/ssd/apps/cratedrop/db/cratedrop.sqlite

# Example queries
SELECT email, role FROM users;
SELECT title, artist FROM tracks LIMIT 10;
```

## Features

- User authentication (JWT)
- Track upload and streaming
- Playlist management (crates)
- Album art support
- SoundCloud likes sync (requires OAuth token)

## API Endpoints

Base URL: `/api`

- `POST /api/auth/login` - Login
- `POST /api/auth/register` - Register
- `GET /api/tracks` - List tracks
- `POST /api/tracks` - Upload track
- `GET /api/tracks/:id/stream` - Stream audio
- `GET /api/playlists` - List playlists
- `POST /api/playlists` - Create playlist
- `GET /api/soundcloud/config` - Get SoundCloud sync config
- `POST /api/soundcloud/sync` - Trigger SoundCloud sync
