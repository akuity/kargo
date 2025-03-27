package encoding

import (
	"net/http"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
)

// Determine attempts to determine the encoding of the provided bytes by first
// parsing the provided content type and then, if that fails, by sniffing the
// provided bytes. If the encoding can be determined, it is returned. If it
// cannot be determined, nil is returned.
func Determine(contentType string, bytes []byte) encoding.Encoding {
	parts := strings.Split(
		strings.TrimSpace(
			strings.ToLower(contentType),
		),
		";",
	)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, charsetPrefix) {
			if enc := CharsetTo(strings.TrimPrefix(part, charsetPrefix)); enc != nil {
				return enc
			}
		}
	}
	// If we get to here, we could not determine the encoding. Try to determine it
	// by sniffing the provided bytes.
	//
	// Note: This actually has little chance of succeeding because it relies
	// heavily on BOMs, which are not commonly included in an HTTP response body.
	// The likely outcome is the function's default return value of
	// application/octet-stream.
	detectedContentType := http.DetectContentType(bytes)
	parts = strings.Split(
		strings.TrimSpace(
			strings.ToLower(detectedContentType),
		),
		";",
	)
	for _, part := range parts[1:] {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, charsetPrefix) {
			// Return no matter whether we determined the encoding or not.
			return CharsetTo(strings.TrimPrefix(part, charsetPrefix))
		}
	}
	// If we get to here, we could not determine the encoding.
	return nil
}

const charsetPrefix = "charset="

// CharsetTo maps common character set names to their corresponding encoding.
func CharsetTo(cs string) encoding.Encoding {
	switch strings.TrimPrefix(cs, charsetPrefix) {
	case "iso-8859-1":
		return charmap.ISO8859_1
	case "utf-8":
		return unicode.UTF8
	case "utf-16", "utf-16be":
		return unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
	case "utf-16le":
		return unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	// TODO: Add other charsets as needed
	default:
		return nil
	}
}

// HasUTF16BOM returns a boolean indicating whether the provided bytes begin
// with either of a UTF-16 Big Endian or Little Endian byte order mark.
func HasUTF16BOM(bytes []byte) bool {
	return len(bytes) >= 2 &&
		(bytes[0] == 0xfe && bytes[1] == 0xff || bytes[0] == 0xff && bytes[1] == 0xfe)
}
