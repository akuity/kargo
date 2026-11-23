package targetstatuses

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseLimit(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		raw      string
		expected int32
		invalid  bool
	}{
		{raw: "", expected: defaultPageSize},
		{raw: "1", expected: 1},
		{raw: "500", expected: maxPageSize},
		{raw: "501", expected: maxPageSize},
		{raw: "0", invalid: true},
		{raw: "-1", invalid: true},
		{raw: "many", invalid: true},
		{raw: "99999999999", invalid: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.raw, func(t *testing.T) {
			t.Parallel()
			limit, err := parseLimit(testCase.raw)
			if testCase.invalid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, testCase.expected, limit)
		})
	}
}

func TestTokens(t *testing.T) {
	t.Parallel()
	cursor := promotionCursor{
		CreatedAt: time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
		ID:        "id-1",
	}
	token := encodeToken(cursor)
	// A token travels in a query string as it is.
	require.Equal(t, token, url.QueryEscape(token))
	var decoded promotionCursor
	require.NoError(t, decodeToken(token, &decoded))
	require.Equal(t, cursor, decoded)

	for _, bad := range []string{"not base64!", "bm90IGpzb24"} {
		require.Error(t, decodeToken(bad, &decoded), bad)
	}
}
