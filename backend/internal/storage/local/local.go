package local

import (
    "context"
    "fmt"
    "io"
    "mime"
    "os"
    "path/filepath"

    "github.com/faraz525/home-music-server/backend/internal/storage"
)

type LocalStorage struct {
    dataDir string
}

func New(dataDir string) *LocalStorage { return &LocalStorage{dataDir: dataDir} }

func (s *LocalStorage) Save(ctx context.Context, userID, trackID, originalName string, r io.Reader) (string, int64, string, error) {
    ext := filepath.Ext(originalName)
    filename := fmt.Sprintf("%s%s", trackID, ext)
    relPath := filepath.Join("library", fmt.Sprintf("user_%s", userID), fmt.Sprintf("track_%s", trackID), filename)
    fullPath := filepath.Join(s.dataDir, relPath)
    if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
        return "", 0, "", err
    }
    f, err := os.Create(fullPath + ".tmp")
    if err != nil {
        return "", 0, "", err
    }
    n, err := io.Copy(f, r)
    cerr := f.Close()
    if err != nil {
        os.Remove(fullPath + ".tmp")
        if cerr == nil { cerr = err }
        return "", 0, "", cerr
    }
    if err := os.Rename(fullPath+".tmp", fullPath); err != nil {
        return "", 0, "", err
    }
    ctype := mime.TypeByExtension(ext)
    if ctype == "" { ctype = "application/octet-stream" }
    return relPath, n, ctype, nil
}

func (s *LocalStorage) Open(ctx context.Context, filePath string) (storage.ReadSeekCloser, storage.FileInfo, error) {
    full := filepath.Join(s.dataDir, filePath)
    f, err := os.Open(full)
    if err != nil { return nil, storage.FileInfo{}, err }
    st, err := f.Stat()
    if err != nil { f.Close(); return nil, storage.FileInfo{}, err }
    return f, storage.FileInfo{Size: st.Size()}, nil
}

func (s *LocalStorage) Delete(ctx context.Context, filePath string) error {
    full := filepath.Join(s.dataDir, filePath)
    if err := os.Remove(full); err != nil && !os.IsNotExist(err) { return err }
    return nil
}

func (s *LocalStorage) ResolveFullPath(filePath string) (string, bool) {
    return filepath.Join(s.dataDir, filePath), true
}

