package analysis

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Result is the structured output of an essentia analysis run.
type Result struct {
	BPM           float64
	BPMConfidence float64 // normalized to [0, 1]
	Key           string  // Camelot notation, e.g. "8A"; "" if unknown
	KeyConfidence float64 // normalized to [0, 1]; 0 if key unknown
}

type rawEssentia struct {
	Rhythm struct {
		BPM           float64 `json:"bpm"`
		BPMConfidence float64 `json:"bpm_confidence"`
	} `json:"rhythm"`
	Tonal struct {
		KeyKey      string  `json:"key_key"`
		KeyScale    string  `json:"key_scale"`
		KeyStrength float64 `json:"key_strength"`
	} `json:"tonal"`
}

// ParseEssentiaOutput reads the JSON written by streaming_extractor_music and
// returns a normalized Result. Returns an error for malformed JSON or missing
// BPM (the only truly required field).
func ParseEssentiaOutput(raw []byte) (Result, error) {
	var e rawEssentia
	if err := json.Unmarshal(raw, &e); err != nil {
		return Result{}, fmt.Errorf("parse essentia json: %w", err)
	}
	if e.Rhythm.BPM <= 0 {
		return Result{}, errors.New("essentia output missing bpm")
	}
	camelot := ToCamelot(e.Tonal.KeyKey, e.Tonal.KeyScale)
	keyConf := normalizeConfidence(e.Tonal.KeyStrength)
	if camelot == "" {
		keyConf = 0
	}
	return Result{
		BPM:           e.Rhythm.BPM,
		BPMConfidence: normalizeConfidence(e.Rhythm.BPMConfidence / 5.0),
		Key:           camelot,
		KeyConfidence: keyConf,
	}, nil
}

func normalizeConfidence(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}
