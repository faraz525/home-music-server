package metadata

import "context"

type AudioMetadata struct {
    DurationSeconds float64
    Title           *string
    Artist          *string
    Album           *string
    Genre           *string
    Year            *int
    SampleRate      *int
    Bitrate         *int
}

type Extractor interface {
    Extract(ctx context.Context, fullPath string) (*AudioMetadata, error)
}

