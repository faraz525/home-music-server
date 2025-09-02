# 🎵 CrateDrop - Private Music Server

**A minimalist, Raspberry Pi-friendly music server for DJs and music enthusiasts**

CrateDrop is a self-hosted web application that lets you securely upload, organize, and stream your music library. It provides a private "Dropbox for tracks" with a sleek web player, perfect for Raspberry Pi deployments with SSD storage.

![CrateDrop](https://img.shields.io/badge/version-v0.1.0-blue)
![License](https://img.shields.io/badge/license-MIT-green)
![Platform](https://img.shields.io/badge/platform-Raspberry%20Pi-orange)

## ✨ Features

- 🔐 **Invite-only access** - Secure, private music sharing
- 📤 **Drag & drop uploads** - Support for WAV, AIFF, FLAC, MP3
- 🔍 **Smart search** - Find tracks by filename, title, or artist
- 🎵 **Web player** - Stream with seek support and playback controls
- 👥 **Multi-user** - Admin panel for user management
- 📱 **Mobile-friendly** - Works great on phones and tablets
- 🚀 **Raspberry Pi optimized** - Low resource usage, SSD storage support

## 🚀 Quick Start

### Prerequisites

- **Raspberry Pi** (4 or 5 recommended) with **64-bit OS**
- **SSD storage** (recommended for music library)
- **Docker & Docker Compose** installed
- **Git** for cloning the repository

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/yourusername/home-music-server.git
   cd home-music-server
   ```

2. **Prepare your storage**
   ```bash
   # Create music directory on your SSD
   sudo mkdir -p /mnt/music/cratedrop
   sudo chown -R $USER:$USER /mnt/music/cratedrop
   ```

3. **Configure environment (optional)**
   ```bash
   # Create and edit environment file
   cat > .env << 'EOF'
   # Application Environment
   APP_ENV=production

   # Base URL (change to your domain/IP)
   BASE_URL=http://localhost

   # Data Directory (where music files and database are stored)
   DATA_DIR=/mnt/music/cratedrop

   # JWT Security (CHANGE THESE IN PRODUCTION!)
   JWT_SECRET=your-super-secret-jwt-key-here
   REFRESH_SECRET=your-super-secret-refresh-key-here

   # Gin Framework Mode (debug/release)
   GIN_MODE=release
   EOF

   nano .env  # Edit JWT secrets and other settings
   ```

4. **Start the application**
   ```bash
   # Build and start all services
   docker compose up -d --build

   # Check that everything is running
   docker compose ps
   ```

5. **Access your music server**
   - Open your browser and go to `http://your-pi-ip`
   - Default admin credentials will be shown in the logs

### First Time Setup

1. **Create admin user**
   ```bash
   docker compose logs backend | grep "Admin user created"
   ```

2. **Login** with the displayed admin credentials

3. **Generate invite codes** from the Admin panel

4. **Share invites** with friends to let them create accounts

## 📖 Usage Guide

### For DJs/Music Users

1. **Sign up** using an invite code from an admin
2. **Login** to access your personal dashboard
3. **Upload tracks** by dragging files or clicking upload
4. **Search & browse** your music library
5. **Play tracks** directly in your browser with full seek support
6. **Manage your tracks** - delete old files to free up space

### For Admins

1. **Access Admin panel** (available only to admin users)
2. **View all users** and their activity
3. **Generate invite codes** for new users
4. **Monitor storage usage** and system health
5. **Manage the music library** - delete any track if needed

## 🔍 Logging & Monitoring

CrateDrop provides comprehensive logging to help you monitor your music server:

### View Logs

```bash
# View all logs
docker compose logs

# Real-time log monitoring
docker compose logs -f

# View specific service logs
docker compose logs backend
docker compose logs frontend

# Recent logs with timestamps
docker compose logs --tail=20 -t

# Logs from last hour
docker compose logs --since 1h
```

### Log Analysis

```bash
# Filter for errors
docker compose logs | grep -i error

# View API requests
docker compose logs backend | grep "GET\|POST"

# Monitor user activity
docker compose logs | grep "192.168.1.100"  # Replace with user IP

# Track file uploads
docker compose logs backend | grep "upload"
```

### Log Files

- **Container logs**: Available via `docker compose logs`
- **File logs**: Stored in `./logs/` directory (if configured)
- **Health checks**: Automatic every 30 seconds

### Log Format

```
[CrateDrop] 2025/09/02 15:04:05 | 200 | 45.23µs | 192.168.1.100 | GET /api/tracks
[GIN] 2025/09/02 - 15:04:05 | 200 | 45.23µs | 192.168.1.100 | GET /api/healthz
```

## 🔧 Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_ENV` | `production` | Environment (development/production) |
| `BASE_URL` | `http://localhost` | Base URL for the application |
| `DATA_DIR` | `/mnt/music/cratedrop` | Directory for music storage |
| `JWT_SECRET` | `dev-jwt-secret` | Secret for JWT tokens |
| `REFRESH_SECRET` | `dev-refresh-secret` | Secret for refresh tokens |

### Storage Layout

```
/mnt/music/cratedrop/
├── library/          # User track storage
│   └── <user_id>/
│       └── <track_id>/
│           └── original.<ext>
├── db/              # SQLite database
├── backups/         # Database backups
└── logs/            # Application logs
```

## 🐛 Troubleshooting

### Common Issues

**Port 80 already in use**
```bash
# Check what's using port 80
sudo netstat -tlnp | grep :80

# Stop conflicting service or change port in docker-compose.yml
# ports:
#   - "8080:80"  # Change from 80 to 8080
```

**Backend container keeps restarting**
```bash
# Check backend logs for errors
docker compose logs backend

# Common causes:
# - Database migration errors
# - Storage permission issues
# - Port conflicts
```

**Cannot upload large files**
```bash
# Check nginx client_max_body_size in nginx.conf
# Default is 2GB, increase if needed
client_max_body_size 4G;
```

**Storage permission errors**
```bash
# Ensure proper permissions on music directory
sudo chown -R 1000:1000 /mnt/music/cratedrop
```

### Health Checks

```bash
# Check application health
curl http://localhost/api/healthz

# Check container status
docker compose ps

# Restart services
docker compose restart
```

## 📡 API Reference

### Authentication Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/auth/signup?invite=CODE` | User registration |
| `POST` | `/api/auth/login` | User login |
| `POST` | `/api/auth/refresh` | Refresh access token |
| `POST` | `/api/auth/logout` | Logout user |
| `GET` | `/api/me` | Get current user info |

### Track Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/tracks` | Upload new track |
| `GET` | `/api/tracks` | List tracks (with search/pagination) |
| `GET` | `/api/tracks/:id` | Get track metadata |
| `GET` | `/api/tracks/:id/stream` | Stream track audio |
| `DELETE` | `/api/tracks/:id` | Delete track |

### Admin Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/users` | List all users (admin only) |
| `POST` | `/api/invites` | Create invite code (admin only) |
| `GET` | `/api/invites` | List invites (admin only) |

## 🏗️ Development

### Prerequisites

- **Go 1.23+** for backend development
- **Node.js 20+** for frontend development
- **Docker & Docker Compose** for containerized development

### Local Development

```bash
# Backend development
cd backend
go mod download
go run main.go

# Frontend development
cd frontend
npm install
npm run dev

# Full stack with Docker
docker compose -f docker-compose.dev.yml up
```

### Building

```bash
# Build all services
docker compose build

# Build specific service
docker compose build backend
docker compose build frontend
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/yourusername/home-music-server/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/home-music-server/discussions)
- **Documentation**: Check the `docs/` directory for detailed guides

## 🙏 Acknowledgments

- Built with [Go](https://golang.org/) and [Gin](https://gin-gonic.com/)
- Frontend powered by [React](https://reactjs.org/) and [Vite](https://vitejs.dev/)
- Styled with [Tailwind CSS](https://tailwindcss.com/)
- Audio processing with [FFmpeg](https://ffmpeg.org/)

---

**Happy mixing! 🎧**