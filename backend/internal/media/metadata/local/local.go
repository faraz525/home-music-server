package local

import (
	"context"
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"

	"github.com/faraz525/home-music-server/backend/internal/media/metadata"
)

type FFProbeExtractor struct{}

func New() *FFProbeExtractor { return &FFProbeExtractor{} }

// ffprobe minimal JSON structures we care about
type ffprobeFormat struct {
	Tags     map[string]any `json:"tags"`
	Duration string         `json:"duration"`
	BitRate  string         `json:"bit_rate"`
}

type ffprobeStream struct {
	CodecType  string `json:"codec_type"`
	SampleRate string `json:"sample_rate"`
	BitRate    string `json:"bit_rate"`
}

type ffprobeOutput struct {
	Format  ffprobeFormat   `json:"format"`
	Streams []ffprobeStream `json:"streams"`
}

func (e *FFProbeExtractor) Extract(ctx context.Context, fullPath string) (*metadata.AudioMetadata, error) {
	// Ask ffprobe for both format and streams in JSON
	out, err := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		fullPath,
	).Output()
	if err != nil {
		return &metadata.AudioMetadata{}, nil
	}

	var probe ffprobeOutput
	if err := json.Unmarshal(out, &probe); err != nil {
		return &metadata.AudioMetadata{}, nil
	}

	md := &metadata.AudioMetadata{}

	// duration
	if d, err := strconv.ParseFloat(strings.TrimSpace(probe.Format.Duration), 64); err == nil && d > 0 {
		md.DurationSeconds = d
	}

	// bitrate: prefer stream audio bitrate, fallback to container
	if len(probe.Streams) > 0 {
		for _, s := range probe.Streams {
			if s.CodecType == "audio" {
				if br, err := strconv.Atoi(strings.TrimSpace(s.BitRate)); err == nil && br > 0 {
					v := br
					md.Bitrate = &v
				}
				if sr, err := strconv.Atoi(strings.TrimSpace(s.SampleRate)); err == nil && sr > 0 {
					v := sr
					md.SampleRate = &v
				}
				break
			}
		}
	}
	if md.Bitrate == nil {
		if br, err := strconv.Atoi(strings.TrimSpace(probe.Format.BitRate)); err == nil && br > 0 {
			v := br
			md.Bitrate = &v
		}
	}

	// tags
	tags := probe.Format.Tags
	if tags != nil {
		if v, ok := tags["title"].(string); ok && strings.TrimSpace(v) != "" {
			s := strings.TrimSpace(v)
			md.Title = &s
		}
		if v, ok := tags["artist"].(string); ok && strings.TrimSpace(v) != "" {
			s := strings.TrimSpace(v)
			md.Artist = &s
		}
		if v, ok := tags["album"].(string); ok && strings.TrimSpace(v) != "" {
			s := strings.TrimSpace(v)
			md.Album = &s
		}
		if v, ok := tags["genre"].(string); ok && strings.TrimSpace(v) != "" {
			s := strings.TrimSpace(v)
			md.Genre = &s
		}
		// Year: try common fields
		for _, key := range []string{"date", "year", "creation_time"} {
			if raw, ok := tags[key]; ok {
				if str, ok := raw.(string); ok {
					str = strings.TrimSpace(str)
					// Take first 4 digits if present
					if len(str) >= 4 {
						if y, err := strconv.Atoi(str[:4]); err == nil && y > 0 {
							v := y
							md.Year = &v
							break
						}
					}
				}
			}
		}
	}

	return md, nil
}
