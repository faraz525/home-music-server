package storage

import (
    "context"
    "io"
)

type ReadSeekCloser interface {
    io.Reader
    io.Seeker
    io.Closer
}

type FileInfo struct {
    Size int64
}

type Storage interface {
    Save(ctx context.Context, userID, trackID, originalName string, r io.Reader) (filePath string, size int64, contentType string, err error)
    Open(ctx context.Context, filePath string) (ReadSeekCloser, FileInfo, error)
    Delete(ctx context.Context, filePath string) error
    ResolveFullPath(filePath string) (string, bool)
}

