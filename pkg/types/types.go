package types

import "strconv"

// MustParseBool parses a string into a boolean value, panicking if the string
// cannot be parsed as a boolean.
func MustParseBool(s string) bool {
	b, err := strconv.ParseBool(s)
	if err != nil {
		panic(err)
	}
	return b
}

// MustParseFloat32 parses a string into a float32 value, panicking if the
// string cannot be parsed as a float32.
func MustParseFloat32(s string) float32 {
	f, err := strconv.ParseFloat(s, 32)
	if err != nil {
		panic(err)
	}
	return float32(f)
}

// MustParseInt parses a string into an int value, panicking if the string
// cannot be parsed as an int.
func MustParseInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return i
}
