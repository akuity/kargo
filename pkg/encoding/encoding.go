package encoding

import (
	"bytes"
	"mime"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
)

const (
	CharsetUTF8    = "utf-8"
	CharsetUTF16   = "utf-16"
	CharsetUTF16BE = "utf-16be"
	CharsetUTF16LE = "utf-16le"
	CharsetISO8859 = "iso-8859-1"
)

var (
	UTF16BEBOM = []byte{0xFE, 0xFF}
	UTF16LEBOM = []byte{0xFF, 0xFE}
)

// GetEncodingFromContentType extracts and returns the encoding from a
// Content-Type header. Returns nil if no charset is specified or the charset is
// unknown.
func GetEncodingFromContentType(contentType string) encoding.Encoding {
	if contentType == "" {
		return nil
	}

	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil
	}

	charset, ok := params["charset"]
	if !ok {
		return nil
	}

	return GetEncodingFromCharset(charset)
}

// GetEncodingFromCharset returns the encoding for a given charset name.
// Returns nil if the charset is unknown.
func GetEncodingFromCharset(charset string) encoding.Encoding {
	switch strings.ToLower(charset) {
	case CharsetISO8859:
		return charmap.ISO8859_1
	case CharsetUTF8:
		return unicode.UTF8
	case CharsetUTF16, CharsetUTF16BE:
		return unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
	case CharsetUTF16LE:
		return unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	default:
		return nil
	}
}

// DetectEncoding attempts to determine the encoding of data.
// It first checks the Content-Type header, then falls back to examining the
// data. If no encoding can be determined, it returns UTF-8 as the default.
func DetectEncoding(contentType string, data []byte) encoding.Encoding {
	// Try to get encoding from Content-Type header
	if enc := GetEncodingFromContentType(contentType); enc != nil {
		return enc
	}

	// Try to detect from BOM if present
	if len(data) >= 2 {
		switch {
		case bytes.Equal(data[:2], UTF16BEBOM):
			return unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
		case bytes.Equal(data[:2], UTF16LEBOM):
			return unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
		}
	}

	// Default to UTF-8
	return unicode.UTF8
}

// HasUTF16BOM returns true if the data begins with a UTF-16 byte order mark.
func HasUTF16BOM(data []byte) bool {
	return len(data) >= 2 && (bytes.Equal(data[:2], UTF16BEBOM) ||
		bytes.Equal(data[:2], UTF16LEBOM))
}
