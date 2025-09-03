package utils

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/faraz525/home-music-server/backend/models"
)

// AudioMetadata represents metadata extracted from an audio file
type AudioMetadata struct {
	Title      string  `json:"title"`
	Artist     string  `json:"artist"`
	Album      string  `json:"album"`
	Genre      string  `json:"genre"`
	Year       int     `json:"year"`
	Duration   float64 `json:"duration"`
	SampleRate int     `json:"sample_rate"`
	Bitrate    int     `json:"bitrate"`
}

// FFProbeOutput represents the structure of ffprobe JSON output
type FFProbeOutput struct {
	Format FFProbeFormat `json:"format"`
}

type FFProbeFormat struct {
	Tags     map[string]interface{} `json:"tags"`
	Duration string                 `json:"duration"`
	BitRate  string                 `json:"bit_rate"`
}

type FFProbeStream struct {
	SampleRate string `json:"sample_rate"`
}

// ExtractMetadata extracts metadata from an audio file using ffprobe
func ExtractMetadata(filePath string) (*AudioMetadata, error) {
	// Run ffprobe command to extract metadata
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath)

	fmt.Printf("[CrateDrop] Extracting metadata from: %s\n", filePath)
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("[CrateDrop] ffprobe failed for %s: %v\n", filePath, err)
		return nil, fmt.Errorf("failed to run ffprobe: %w", err)
	}
	fmt.Printf("[CrateDrop] ffprobe output received for %s\n", filePath)

	var probeData struct {
		Format  FFProbeFormat   `json:"format"`
		Streams []FFProbeStream `json:"streams"`
	}

	if err := json.Unmarshal(output, &probeData); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	metadata := &AudioMetadata{}

	// Extract duration
	if duration, err := strconv.ParseFloat(probeData.Format.Duration, 64); err == nil {
		metadata.Duration = duration
	}

	// Extract bitrate
	if bitrate, err := strconv.Atoi(probeData.Format.BitRate); err == nil {
		metadata.Bitrate = bitrate
	}

	// Extract sample rate from first audio stream
	if len(probeData.Streams) > 0 {
		if sampleRate, err := strconv.Atoi(probeData.Streams[0].SampleRate); err == nil {
			metadata.SampleRate = sampleRate
		}
	}

	// Extract metadata tags
	tags := probeData.Format.Tags
	if tags != nil {
		// Title
		if title, ok := tags["title"]; ok {
			if titleStr, ok := title.(string); ok {
				metadata.Title = strings.TrimSpace(titleStr)
			}
		}

		// Artist
		if artist, ok := tags["artist"]; ok {
			if artistStr, ok := artist.(string); ok {
				metadata.Artist = strings.TrimSpace(artistStr)
			}
		}

		// Album
		if album, ok := tags["album"]; ok {
			if albumStr, ok := album.(string); ok {
				metadata.Album = strings.TrimSpace(albumStr)
			}
		}

		// Genre
		if genre, ok := tags["genre"]; ok {
			if genreStr, ok := genre.(string); ok {
				metadata.Genre = strings.TrimSpace(genreStr)
			}
		}

		// Year (try different tag names)
		yearTags := []string{"date", "year", "creation_time"}
		for _, tag := range yearTags {
			if yearVal, ok := tags[tag]; ok {
				var yearStr string
				if str, ok := yearVal.(string); ok {
					yearStr = str
				} else {
					continue
				}

				// Handle different date formats
				if strings.Contains(yearStr, "-") {
					// ISO date format like "2023-01-01"
					if t, err := time.Parse("2006-01-02", yearStr[:10]); err == nil {
						metadata.Year = t.Year()
						break
					}
				} else if len(yearStr) >= 4 {
					// Year only format like "2023"
					if year, err := strconv.Atoi(yearStr[:4]); err == nil {
						metadata.Year = year
						break
					}
				}
			}
		}
	}

	return metadata, nil
}

// CreateTrackFromMetadata creates a Track model from extracted metadata and request data
func CreateTrackFromMetadata(metadata *AudioMetadata, userID, originalFilename, contentType, filePath string, sizeBytes int64, req *models.UploadTrackRequest) *models.Track {
	fmt.Printf("[CrateDrop] Creating track from metadata: %+v\n", metadata)

	track := &models.Track{
		OwnerUserID:      userID,
		OriginalFilename: originalFilename,
		ContentType:      contentType,
		SizeBytes:        sizeBytes,
		FilePath:         filePath,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Use metadata values as defaults, but allow request to override
	if metadata.Duration > 0 {
		track.DurationSeconds = &metadata.Duration
	}

	// Title: use request if provided, otherwise metadata
	if req.Title != "" {
		track.Title = StringToPtr(req.Title)
	} else if metadata.Title != "" {
		track.Title = StringToPtr(metadata.Title)
	}

	// Artist: use request if provided, otherwise metadata
	if req.Artist != "" {
		track.Artist = StringToPtr(req.Artist)
	} else if metadata.Artist != "" {
		track.Artist = StringToPtr(metadata.Artist)
	}

	// Album: use request if provided, otherwise metadata
	if req.Album != "" {
		track.Album = StringToPtr(req.Album)
	} else if metadata.Album != "" {
		track.Album = StringToPtr(metadata.Album)
	}

	// Genre: use request if provided, otherwise metadata
	if req.Genre != "" {
		track.Genre = StringToPtr(req.Genre)
	} else if metadata.Genre != "" {
		track.Genre = StringToPtr(metadata.Genre)
	}

	// Year: use request if provided, otherwise metadata
	if req.Year > 0 {
		track.Year = &req.Year
	} else if metadata.Year > 0 {
		track.Year = &metadata.Year
	}

	// Sample rate: use request if provided, otherwise metadata
	if req.SampleRate > 0 {
		track.SampleRate = &req.SampleRate
	} else if metadata.SampleRate > 0 {
		track.SampleRate = &metadata.SampleRate
	}

	// Bitrate: use request if provided, otherwise metadata
	if req.Bitrate > 0 {
		track.Bitrate = &req.Bitrate
	} else if metadata.Bitrate > 0 {
		track.Bitrate = &metadata.Bitrate
	}

	return track
}
