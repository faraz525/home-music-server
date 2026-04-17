// Package tracksql holds SQL helpers shared between the tracks and
// playlists data layers. It lives in internal/ to avoid an import cycle:
// the tracks package already imports playlists (via its HTTP handlers),
// so the reverse import needed by playlists.Repository goes through
// this leaf package instead of the tracks package directly.
package tracksql

import (
	imodels "github.com/faraz525/home-music-server/backend/internal/models"
)

// TrackColumns is the canonical column list for SELECT statements that
// scan into imodels.Track via ScanTrack. Columns are prefixed with the
// alias "t" so the constant can be used in both single-table and JOINed
// queries; single-table queries must alias tracks as t.
const TrackColumns = "t.id, t.owner_user_id, t.original_filename, t.content_type, t.size_bytes, " +
	"t.duration_seconds, t.title, t.artist, t.album, t.genre, t.year, " +
	"t.sample_rate, t.bitrate, t.file_path, t.created_at, t.updated_at"

// RowScanner is the minimal interface implemented by *sql.Row and *sql.Rows.
type RowScanner interface {
	Scan(dest ...any) error
}

// ScanTrack scans one row of TrackColumns into a Track.
func ScanTrack(row RowScanner) (*imodels.Track, error) {
	var t imodels.Track
	if err := row.Scan(
		&t.ID, &t.OwnerUserID, &t.OriginalFilename, &t.ContentType, &t.SizeBytes,
		&t.DurationSeconds, &t.Title, &t.Artist, &t.Album, &t.Genre, &t.Year,
		&t.SampleRate, &t.Bitrate, &t.FilePath, &t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &t, nil
}
