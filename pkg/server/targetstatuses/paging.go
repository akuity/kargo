package targetstatuses

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	libhttp "github.com/akuity/kargo/pkg/http"
)

// defaultPageSize is how many items a list returns when no limit is asked
// for, and maxPageSize the most it returns whatever limit is.
const (
	defaultPageSize = 100
	maxPageSize     = 500
)

// parseLimit reads a page size: defaultPageSize when none is given, and
// never more than maxPageSize.
func parseLimit(raw string) (int32, error) {
	if raw == "" {
		return defaultPageSize, nil
	}
	limit, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || limit <= 0 {
		return 0, libhttp.ErrorStr(
			fmt.Sprintf("invalid limit %q", raw),
			http.StatusBadRequest,
		)
	}
	return int32(min(limit, maxPageSize)), nil
}

// cursor is what a continue token holds: a place in a list.
type cursor interface {
	// complete reports whether the cursor names a place. One that does not
	// came from no page, so the token it was read from is refused.
	complete() bool
}

// encodeToken returns a cursor as a continue token. The token is opaque to
// callers, who only hand it back.
func encodeToken(c cursor) (string, error) {
	raw, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("error encoding continue token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// decodeToken reads a continue token back into the cursor it was made from.
// A token that does not decode, or that decodes to a cursor naming no place,
// is a bad request.
func decodeToken(token string, c cursor) error {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err == nil {
		err = json.Unmarshal(raw, c)
	}
	if err != nil || !c.complete() {
		return libhttp.ErrorStr(
			fmt.Sprintf("invalid continue token %q", token),
			http.StatusBadRequest,
		)
	}
	return nil
}
