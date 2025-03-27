package encoding

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
)

func TestDetermine(t *testing.T) {
	testUTF16NoBOM := []byte{0x00, 0x48, 0x00, 0x65, 0x00, 0x6c, 0x00, 0x6c, 0x00, 0x6f}
	testUTF16WithBEBOM := []byte{0xfe, 0xff, 0x00, 0x48, 0x00, 0x65, 0x00, 0x6c, 0x00, 0x6c, 0x00, 0x6f}
	testCases := []struct {
		name        string
		contentType string
		bytes       []byte
		expected    encoding.Encoding
	}{
		{
			name:        "charset specified in content type",
			contentType: "text/plain; charset=iso-8859-1",
			expected:    charmap.ISO8859_1,
		},
		{
			name:        "charset not specified in content type; cannot be inferred from bytes",
			contentType: "",
			// With no BOM, we cannot determine the encoding.
			bytes:    testUTF16NoBOM,
			expected: nil,
		},
		{
			name:        "charset not specified in content type; can be inferred from bytes",
			contentType: "",
			bytes:       testUTF16WithBEBOM,
			expected:    unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expected, Determine(testCase.contentType, testCase.bytes))
		})
	}
}

func TestCharsetTo(t *testing.T) {
	testCases := []struct {
		charset  string
		expected encoding.Encoding
	}{
		{
			charset:  "iso-8859-1",
			expected: charmap.ISO8859_1,
		},
		{
			charset:  "utf-8",
			expected: unicode.UTF8,
		},
		{
			charset:  "utf-16",
			expected: unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM),
		},
		{
			charset:  "utf-16be",
			expected: unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM),
		},
		{
			charset:  "utf-16le",
			expected: unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM),
		},
		{
			charset:  "other",
			expected: nil,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.charset, func(t *testing.T) {
			require.Equal(t, testCase.expected, CharsetTo(testCase.charset))
		})
	}
}

func TestHasUTF16BOM(t *testing.T) {
	testCases := []struct {
		name     string
		bytes    []byte
		expected bool
	}{
		{
			name:     "empty",
			expected: false,
		},
		{
			name:     "UTF-16BE BOM",
			bytes:    []byte{0xfe, 0xff},
			expected: true,
		},
		{
			name:     "UTF-16LE BOM",
			bytes:    []byte{0xff, 0xfe},
			expected: true,
		},
		{
			name:     "UTF-8 BOM",
			bytes:    []byte{0xef, 0xbb, 0xbf},
			expected: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expected, HasUTF16BOM(testCase.bytes))
		})
	}
}
