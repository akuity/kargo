package handler

import (
	"embed"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed testdata/*
var testData embed.FS

func Test_ValidateTestData(t *testing.T) {
	entries, err := testData.ReadDir("testdata")
	require.NoError(t, err)
	require.NotEmpty(t, entries)
}
