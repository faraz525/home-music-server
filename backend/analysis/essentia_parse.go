package analysis

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

// essentiaBPMConfidenceScale is the approximate upper bound of essentia's
// rhythmextractor2013 bpm_confidence metric (a histogram-peak score, not a
// probability). Dividing by this scale and clamping maps it into [0, 1].
// Ref: https://essentia.upf.edu/reference/streaming_RhythmExtractor2013.html
const essentiaBPMConfidenceScale = 5.0

// Result is the structured output of an essentia analysis run.
type Result struct {
	BPM           float64
	BPMConfidence float64 // in [0, 1]
	Key           string  // Camelot notation, e.g. "8A"; "" if unknown
	KeyConfidence float64 // in [0, 1]; 0 if key unknown
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
// returns a normalized Result. Returns an error for malformed JSON or a
// missing / non-positive / NaN BPM (the only truly required field).
func ParseEssentiaOutput(raw []byte) (Result, error) {
	var e rawEssentia
	if err := json.Unmarshal(raw, &e); err != nil {
		return Result{}, fmt.Errorf("parse essentia json: %w", err)
	}
	if math.IsNaN(e.Rhythm.BPM) || e.Rhythm.BPM <= 0 {
		return Result{}, errors.New("essentia output missing or non-positive bpm")
	}
	camelot := ToCamelot(e.Tonal.KeyKey, e.Tonal.KeyScale)
	keyConf := clamp01(e.Tonal.KeyStrength)
	if camelot == "" {
		keyConf = 0
	}
	return Result{
		BPM:           e.Rhythm.BPM,
		BPMConfidence: clamp01(e.Rhythm.BPMConfidence / essentiaBPMConfidenceScale),
		Key:           camelot,
		KeyConfidence: keyConf,
	}, nil
}

// clamp01 coerces x into [0, 1], treating NaN as 0.
func clamp01(x float64) float64 {
	if math.IsNaN(x) || x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}
