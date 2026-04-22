package analysis

var majorToCamelot = map[string]string{
	"C": "8B", "G": "9B", "D": "10B", "A": "11B", "E": "12B", "B": "1B",
	"F#": "2B", "Gb": "2B",
	"Db": "3B", "C#": "3B",
	"Ab": "4B", "G#": "4B",
	"Eb": "5B", "D#": "5B",
	"Bb": "6B", "A#": "6B",
	"F": "7B",
}

var minorToCamelot = map[string]string{
	"A": "8A", "E": "9A", "B": "10A",
	"F#": "11A", "Gb": "11A",
	"C#": "12A", "Db": "12A",
	"G#": "1A", "Ab": "1A",
	"Eb": "2A", "D#": "2A",
	"Bb": "3A", "A#": "3A",
	"F": "4A", "C": "5A", "G": "6A", "D": "7A",
}

// ToCamelot converts a musical key + scale ("A" + "minor") to Camelot wheel
// notation ("8A"). Returns empty string for unrecognized inputs.
//
// The Camelot wheel indexes the 12 major keys as 1B..12B and the 12 minor
// keys as 1A..12A, ordered so that +1/-1 and same-number-other-letter keys
// are harmonically compatible. Scale comparison is case-sensitive: only the
// exact strings "major" and "minor" are recognized.
func ToCamelot(key, scale string) string {
	switch scale {
	case "major":
		return majorToCamelot[key]
	case "minor":
		return minorToCamelot[key]
	default:
		return ""
	}
}
