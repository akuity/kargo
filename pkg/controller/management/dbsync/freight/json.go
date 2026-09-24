package freight

import (
	"bytes"
	"encoding/json"
	"math/big"
	"reflect"
	"strings"
)

// JSONB changes whitespace, object key order, and numeric notation. Compare
// decoded values without converting numbers to float64 and losing precision.
func equalJSON(a, b []byte) bool {
	if len(a) == 0 || len(b) == 0 {
		return len(a) == len(b)
	}
	var left, right any
	decoder := json.NewDecoder(bytes.NewReader(a))
	decoder.UseNumber()
	if err := decoder.Decode(&left); err != nil {
		return false
	}
	decoder = json.NewDecoder(bytes.NewReader(b))
	decoder.UseNumber()
	if err := decoder.Decode(&right); err != nil {
		return false
	}
	return reflect.DeepEqual(normalizeJSON(left), normalizeJSON(right))
}

func normalizeJSON(value any) any {
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			v[key] = normalizeJSON(item)
		}
	case []any:
		for i, item := range v {
			v[i] = normalizeJSON(item)
		}
	case json.Number:
		return normalizeNumber(v)
	}
	return value
}

type decimalNumber struct {
	coefficient string
	exponent    string
}

func normalizeNumber(number json.Number) decimalNumber {
	coefficient, exponentText, _ := strings.Cut(strings.ToLower(string(number)), "e")
	exponent := new(big.Int)
	if exponentText != "" {
		// The decoder has already validated the JSON number's exponent.
		_, _ = exponent.SetString(exponentText, 10)
	}
	negative := strings.HasPrefix(coefficient, "-")
	coefficient = strings.TrimPrefix(coefficient, "-")
	if dot := strings.IndexByte(coefficient, '.'); dot >= 0 {
		exponent.Sub(exponent, big.NewInt(int64(len(coefficient)-dot-1)))
		coefficient = coefficient[:dot] + coefficient[dot+1:]
	}
	coefficient = strings.TrimLeft(coefficient, "0")
	if coefficient == "" {
		return decimalNumber{coefficient: "0", exponent: "0"}
	}
	trimmed := strings.TrimRight(coefficient, "0")
	exponent.Add(exponent, big.NewInt(int64(len(coefficient)-len(trimmed))))
	if negative {
		trimmed = "-" + trimmed
	}
	// Keep the exponent separate instead of allocating a huge power of ten.
	return decimalNumber{coefficient: trimmed, exponent: exponent.String()}
}
