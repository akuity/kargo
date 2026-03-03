package main

import (
	"context"
	"encoding/json"
	"os"
)

func main() {
	releases, err := fetchBestReleases(context.Background(), releasesBaseURL)
	if err != nil {
		panic(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(
		struct {
			Releases []Release `json:"releases"`
		}{
			Releases: releases,
		},
	); err != nil {
		panic(err)
	}
}
