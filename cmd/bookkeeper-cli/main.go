package main

import (
	"fmt"
	"os"
)

func main() {
	cmd, err := newRootCommand()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err = cmd.Execute(); err != nil {
		// Cobra will display the error for us. No need to do it ourselves.
		os.Exit(1)
	}
}
