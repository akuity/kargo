package main

import (
	"fmt"
	"os"

	securejoin "github.com/cyphar/filepath-securejoin"
)

func main() {
	tempDir, err := os.MkdirTemp("", "test-securejoin")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)

	fmt.Printf("TempDir: %s\n", tempDir)

	testPaths := []string{
		"../../../tmp/malicious.yaml",
		"../../etc/passwd",
		"normal/file.yaml",
	}

	for _, testPath := range testPaths {
		result, err := securejoin.SecureJoin(tempDir, testPath)
		fmt.Printf("Path: %q -> Result: %q, Error: %v\n", testPath, result, err)
	}
}
