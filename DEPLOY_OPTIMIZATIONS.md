# Quick Deployment Guide - Performance Optimizations

## 🚀 Quick Deploy (Recommended for Testing Phase)

Since you have few tracks currently, the simplest approach:

```bash
# 1. Stop containers
docker-compose down

# 2. Optional: Backup current data
cp /mnt/music/cratedrop/db/cratedrop.sqlite /mnt/music/cratedrop/db/backup_$(date +%Y%m%d).sqlite

# 3. Remove old database (will be recreated with new schema)
rm /mnt/music/cratedrop/db/cratedrop.sqlite*

# 4. Rebuild backend with new code
docker-compose build backend

# 5. Start everything
docker-compose up -d

# 6. Check logs
docker-compose logs -f backend
```

That's it! The new optimized schema will be created automatically.

## 🧪 Test the Optimizations

### 1. Upload Some Tracks
```bash
# Upload via the web UI or API
curl -X POST http://your-pi:8080/api/tracks \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "file=@your-track.mp3" \
  -F "title=Test Track"
```

### 2. Test Search (FTS5)
```bash
# Should be instant even with thousands of tracks
curl "http://your-pi:8080/api/tracks?q=jazz" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 3. Test Unsorted Crate
```bash
# Should be very fast now (uses LEFT JOIN instead of NOT IN)
curl "http://your-pi:8080/api/tracks?playlist_id=unsorted" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 4. Check Database Stats
```bash
# SSH to your Pi
ssh pi@your-pi

# Check database health
sqlite3 /mnt/music/cratedrop/db/cratedrop.sqlite "SELECT * FROM db_stats;"
```

## 📊 What Changed

### Schema Changes
- ✅ FTS5 full-text search (10-100x faster)
- ✅ Optimized indexes for Raspberry Pi
- ✅ Proper CASCADE DELETE
- ✅ Position field now works correctly
- ✅ Raspberry Pi-specific PRAGMA optimizations

### Code Changes
- ✅ `GetTracksNotInPlaylist` - Uses LEFT JOIN (much faster)
- ✅ `SearchTracks` - Uses FTS5 instead of LIKE
- ✅ `AddTracksToPlaylist` - Populates position correctly
- ✅ `RemoveTracksFromPlaylist` - Reorders positions automatically

## 🔍 Verify Everything Works

```bash
# 1. Check backend is running
docker-compose ps

# 2. Check backend logs for errors
docker-compose logs backend | grep -i error

# 3. Test API health
curl http://your-pi:8080/api/healthz

# 4. Verify FTS5 is working
sqlite3 /mnt/music/cratedrop/db/cratedrop.sqlite \
  "SELECT name FROM sqlite_master WHERE type='table' AND name='tracks_fts';"
# Should return: tracks_fts
```

## 🎯 Performance Benchmarks

After deploying, test with:

```bash
# Time a search query
time curl -s "http://your-pi:8080/api/tracks?q=test" \
  -H "Authorization: Bearer YOUR_TOKEN" > /dev/null

# Time unsorted crate
time curl -s "http://your-pi:8080/api/tracks?playlist_id=unsorted" \
  -H "Authorization: Bearer YOUR_TOKEN" > /dev/null
```

Should complete in < 50ms even with thousands of tracks!

## ⚠️ Troubleshooting

### Issue: Database locked error
```bash
# Check if old connections are open
docker-compose restart backend
```

### Issue: Search returns no results
```bash
# Rebuild FTS5 index
sqlite3 /mnt/music/cratedrop/db/cratedrop.sqlite \
  "INSERT INTO tracks_fts(tracks_fts) VALUES('rebuild');"
```

### Issue: Position field is NULL
```bash
# Fix positions for existing playlists
sqlite3 /mnt/music/cratedrop/db/cratedrop.sqlite << 'EOF'
UPDATE playlist_tracks 
SET position = (
    SELECT COUNT(*) FROM playlist_tracks pt2
    WHERE pt2.playlist_id = playlist_tracks.playlist_id
    AND pt2.rowid < playlist_tracks.rowid
)
WHERE position IS NULL OR position = 0;
EOF
```

## 📈 Expected Performance

| Library Size | Search Time | Unsorted Query | Upload Time |
|--------------|-------------|----------------|-------------|
| 100 tracks   | < 10ms      | < 10ms         | ~5s         |
| 1,000 tracks | < 20ms      | < 15ms         | ~5s         |
| 10,000 tracks| < 50ms      | < 25ms         | ~5s         |

Upload time is dominated by network transfer and ffmpeg metadata extraction.

## 🎉 You're Done!

Your music server is now optimized for Raspberry Pi performance. Enjoy fast searching and smooth operation even with large libraries!

---

**See `PERFORMANCE_OPTIMIZATIONS.md` for detailed technical information.**

