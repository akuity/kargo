package yaml

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// QuoteIfNecessary takes a value and returns it as-is if it is not a string. If
// it is a string, it ascertains whether a YAML parser would interpret it as
// another type. If so, it returns the string with additional quotes around it.
// If not, it returns the string as-is.
func QuoteIfNecessary(val any) any {
	valStr, ok := val.(string)
	if !ok {
		return val
	}
	// If valStr is parseable as a float64, return that. float64 is used because
	// it can represent all JSON numbers.
	//
	// NB: This is attempted prior to attempting to parse valStr as a boolean so
	// that "0" and "1" will be interpreted as numbers.
	if _, err := strconv.ParseFloat(valStr, 64); err == nil {
		return fmt.Sprintf("%q", valStr)
	}
	// If valStr is parseable as a bool return that.
	if _, err := strconv.ParseBool(valStr); err == nil {
		return fmt.Sprintf("%q", valStr)
	}
	// If valStr is valid JSON, return its unmarshaled value. This could be an
	// object, array, or null.
	if json.Valid([]byte(valStr)) {
		return fmt.Sprintf("%q", valStr)
	}
	// If we get to here, just return the string.
	return val
}
