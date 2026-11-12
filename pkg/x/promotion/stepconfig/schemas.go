// Package stepconfig holds the JSON schemas for the configuration of Kargo's
// built-in promotion steps, and the Go types generated from them.
package stepconfig

import "embed"

// Schemas holds the JSON schemas for the configuration of Kargo's built-in
// promotion steps, under the schemas/ directory.
//
//go:embed schemas/*
var Schemas embed.FS
