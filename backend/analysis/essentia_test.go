package analysis

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// writeStubBinary creates a fake streaming_extractor_music in a temp dir that
// writes the supplied JSON to its second argument and exits 0. Returns the
// directory path so the caller can prepend it to PATH.
func writeStubBinary(t *testing.T, outputJSON string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("stub shell script requires a POSIX shell")
	}
	dir := t.TempDir()
	script := "#!/usr/bin/env bash\nprintf '%s' '" + outputJSON + "' > \"$2\"\n"
	path := filepath.Join(dir, "streaming_extractor_music")
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("write stub: %v", err)
	}
	return dir
}

func withPathPrepended(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestAnalyzer_Analyze_Success(t *testing.T) {
	stubDir := writeStubBinary(t, `{"rhythm":{"bpm":124.0,"bpm_confidence":4.0},"tonal":{"key_key":"C","key_scale":"major","key_strength":0.9}}`)
	withPathPrepended(t, stubDir)

	audio := filepath.Join(t.TempDir(), "fake.wav")
	os.WriteFile(audio, []byte("fake"), 0644)

	a := NewAnalyzer(30 * time.Second)
	got, err := a.Analyze(context.Background(), audio)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if got.BPM != 124.0 {
		t.Errorf("BPM = %v, want 124.0", got.BPM)
	}
	if got.Key != "8B" {
		t.Errorf("Key = %q, want 8B", got.Key)
	}
}

func TestAnalyzer_Analyze_BinaryMissing(t *testing.T) {
	t.Setenv("PATH", "/nonexistent")

	a := NewAnalyzer(30 * time.Second)
	_, err := a.Analyze(context.Background(), "/tmp/anything.wav")
	if !errors.Is(err, ErrBinaryMissing) {
		t.Fatalf("want ErrBinaryMissing, got %v", err)
	}
}

func TestAnalyzer_Analyze_FileMissing(t *testing.T) {
	// Stub exits 0 but we want the wrapper to return ErrFileMissing before
	// even calling the binary.
	stubDir := writeStubBinary(t, `{"rhythm":{"bpm":120,"bpm_confidence":2.0},"tonal":{"key_key":"A","key_scale":"minor","key_strength":0.5}}`)
	withPathPrepended(t, stubDir)

	a := NewAnalyzer(30 * time.Second)
	_, err := a.Analyze(context.Background(), "/tmp/definitely-does-not-exist-xyz.wav")
	if !errors.Is(err, ErrFileMissing) {
		t.Fatalf("want ErrFileMissing, got %v", err)
	}
}

func TestAnalyzer_Analyze_MalformedOutput(t *testing.T) {
	stubDir := writeStubBinary(t, `{not json`)
	withPathPrepended(t, stubDir)

	audio := filepath.Join(t.TempDir(), "fake.wav")
	os.WriteFile(audio, []byte("fake"), 0644)

	a := NewAnalyzer(30 * time.Second)
	_, err := a.Analyze(context.Background(), audio)
	if !errors.Is(err, ErrMalformedOutput) {
		t.Fatalf("want ErrMalformedOutput, got %v", err)
	}
}
