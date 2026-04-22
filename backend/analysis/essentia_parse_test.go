package analysis

import (
	"os"
	"testing"
)

func TestParseEssentiaOutput_Success(t *testing.T) {
	raw, err := os.ReadFile("testdata/essentia_success.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	got, err := ParseEssentiaOutput(raw)
	if err != nil {
		t.Fatalf("ParseEssentiaOutput: %v", err)
	}
	if got.BPM != 128.04 {
		t.Errorf("BPM = %v, want 128.04", got.BPM)
	}
	// bpm_confidence of 3.82 should normalize to min(3.82/5.0, 1.0) = 0.764
	if got.BPMConfidence < 0.76 || got.BPMConfidence > 0.77 {
		t.Errorf("BPMConfidence = %v, want ~0.764", got.BPMConfidence)
	}
	if got.Key != "8A" {
		t.Errorf("Key = %q, want %q", got.Key, "8A")
	}
	if got.KeyConfidence != 0.78 {
		t.Errorf("KeyConfidence = %v, want 0.78", got.KeyConfidence)
	}
}

func TestParseEssentiaOutput_MalformedJSON(t *testing.T) {
	_, err := ParseEssentiaOutput([]byte("{not json"))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestParseEssentiaOutput_MissingBPM(t *testing.T) {
	raw := []byte(`{"tonal":{"key_key":"A","key_scale":"minor","key_strength":0.5}}`)
	_, err := ParseEssentiaOutput(raw)
	if err == nil {
		t.Fatal("expected error when BPM is missing")
	}
}

func TestParseEssentiaOutput_UnknownKey_StillReturnsBPM(t *testing.T) {
	// An invalid scale should leave Key empty but BPM still populated.
	raw := []byte(`{"rhythm":{"bpm":120,"bpm_confidence":2.0},"tonal":{"key_key":"C","key_scale":"phrygian","key_strength":0.5}}`)
	got, err := ParseEssentiaOutput(raw)
	if err != nil {
		t.Fatalf("ParseEssentiaOutput: %v", err)
	}
	if got.BPM != 120 {
		t.Errorf("BPM = %v, want 120", got.BPM)
	}
	if got.Key != "" {
		t.Errorf("Key = %q, want empty (unknown scale)", got.Key)
	}
	if got.KeyConfidence != 0 {
		t.Errorf("KeyConfidence = %v, want 0 when key unknown", got.KeyConfidence)
	}
}

func TestParseEssentiaOutput_ConfidenceClamped(t *testing.T) {
	// bpm_confidence > 5.0 should clamp to 1.0
	raw := []byte(`{"rhythm":{"bpm":120,"bpm_confidence":8.0},"tonal":{"key_key":"A","key_scale":"minor","key_strength":1.5}}`)
	got, err := ParseEssentiaOutput(raw)
	if err != nil {
		t.Fatalf("ParseEssentiaOutput: %v", err)
	}
	if got.BPMConfidence != 1.0 {
		t.Errorf("BPMConfidence = %v, want 1.0", got.BPMConfidence)
	}
	if got.KeyConfidence != 1.0 {
		t.Errorf("KeyConfidence = %v, want 1.0", got.KeyConfidence)
	}
}
