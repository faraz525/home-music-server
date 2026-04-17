package analysis

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Sentinel errors so callers (the manager) can decide retry policy.
var (
	ErrBinaryMissing   = errors.New("essentia binary not found on PATH")
	ErrFileMissing     = errors.New("audio file not found")
	ErrTimeout         = errors.New("essentia analysis timed out")
	ErrMalformedOutput = errors.New("essentia produced malformed output")
)

const binaryName = "streaming_extractor_music"

// Analyzer wraps the streaming_extractor_music binary.
type Analyzer struct {
	timeout time.Duration
}

func NewAnalyzer(timeout time.Duration) *Analyzer {
	return &Analyzer{timeout: timeout}
}

// BinaryAvailable reports whether the essentia binary is on PATH. Call once
// at startup to decide whether to start the ticker. Analyze() re-checks, so
// this is an optimization signal only — not a guarantee.
func BinaryAvailable() bool {
	_, err := exec.LookPath(binaryName)
	return err == nil
}

// Analyze runs the essentia extractor on the given audio file path.
func (a *Analyzer) Analyze(ctx context.Context, audioPath string) (Result, error) {
	if _, err := exec.LookPath(binaryName); err != nil {
		return Result{}, ErrBinaryMissing
	}
	if _, err := os.Stat(audioPath); err != nil {
		if os.IsNotExist(err) {
			return Result{}, ErrFileMissing
		}
		return Result{}, fmt.Errorf("stat audio: %w", err)
	}

	outDir, err := os.MkdirTemp("", "essentia-*")
	if err != nil {
		return Result{}, fmt.Errorf("mktemp: %w", err)
	}
	defer os.RemoveAll(outDir)
	// The .json suffix is load-bearing: streaming_extractor_music infers the
	// output format from the extension.
	outPath := filepath.Join(outDir, "out.json")

	runCtx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, binaryName, audioPath, outPath)
	output, err := cmd.CombinedOutput()
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		return Result{}, ErrTimeout
	}
	// Preserve parent-ctx cancellation so the manager can distinguish shutdown
	// from an actual analysis failure.
	if ctx.Err() != nil {
		return Result{}, ctx.Err()
	}
	if err != nil {
		return Result{}, fmt.Errorf("essentia exec failed: %w (output: %s)", err, string(output))
	}

	raw, err := os.ReadFile(outPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Result{}, fmt.Errorf("%w: output file not created (stderr: %s)", ErrMalformedOutput, string(output))
		}
		return Result{}, fmt.Errorf("read essentia output: %w", err)
	}
	result, err := ParseEssentiaOutput(raw)
	if err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrMalformedOutput, err)
	}
	return result, nil
}
