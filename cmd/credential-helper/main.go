package main

import (
	"fmt"
	"os"
)

func main() {
	password := os.Getenv("GIT_PASSWORD")
	if password == "" {
		fmt.Fprintln(os.Stderr, "GIT_PASSWORD must be set")
		os.Exit(1)
	}
	fmt.Println(password)
}
