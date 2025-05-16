package encoding

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
)

func TestGetEncodingFromContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		want        any
	}{
		{
			name:        "empty content type",
			contentType: "",
			want:        nil,
		},
		{
			name:        "no charset",
			contentType: "text/plain",
			want:        nil,
		},
		{
			name:        "utf-8 charset",
			contentType: "text/plain; charset=utf-8",
			want:        unicode.UTF8,
		},
		{
			name:        "utf-16 charset",
			contentType: "text/plain; charset=utf-16",
			want:        unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM),
		},
		{
			name:        "iso-8859-1 charset",
			contentType: "text/plain; charset=iso-8859-1",
			want:        charmap.ISO8859_1,
		},
		{
			name:        "mixed case charset",
			contentType: "text/plain; charset=UTF-8",
			want:        unicode.UTF8,
		},
		{
			name:        "invalid content type",
			contentType: "invalid content type",
			want:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetEncodingFromContentType(tt.contentType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetEncodingFromCharset(t *testing.T) {
	tests := []struct {
		charset string
		want    any
	}{
		{charset: "utf-8", want: unicode.UTF8},
		{charset: "UTF-8", want: unicode.UTF8},
		{charset: "utf-16", want: unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)},
		{charset: "utf-16be", want: unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)},
		{charset: "utf-16le", want: unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)},
		{charset: "iso-8859-1", want: charmap.ISO8859_1},
		{charset: "unknown", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.charset, func(t *testing.T) {
			got := GetEncodingFromCharset(tt.charset)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDetectEncoding(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		data        []byte
		want        any
	}{
		{
			name:        "explicit utf-8 in content type",
			contentType: "text/plain; charset=utf-8",
			data:        []byte{},
			want:        unicode.UTF8,
		},
		{
			name:        "utf-16be BOM",
			contentType: "",
			data:        []byte{0xFE, 0xFF, 0x00, 0x61},
			want:        unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM),
		},
		{
			name:        "utf-16le BOM",
			contentType: "",
			data:        []byte{0xFF, 0xFE, 0x61, 0x00},
			want:        unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM),
		},
		{
			name:        "default to utf-8",
			contentType: "",
			data:        []byte{0x61, 0x62, 0x63},
			want:        unicode.UTF8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectEncoding(tt.contentType, tt.data)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHasUTF16BOM(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{
			name: "empty data",
			data: []byte{},
			want: false,
		},
		{
			name: "too short",
			data: []byte{0xFE},
			want: false,
		},
		{
			name: "utf-16be BOM",
			data: []byte{0xFE, 0xFF, 0x00, 0x61},
			want: true,
		},
		{
			name: "utf-16le BOM",
			data: []byte{0xFF, 0xFE, 0x61, 0x00},
			want: true,
		},
		{
			name: "not a BOM",
			data: []byte{0x61, 0x62},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasUTF16BOM(tt.data)
			assert.Equal(t, tt.want, got)
		})
	}
}
