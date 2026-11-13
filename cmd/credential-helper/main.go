package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

func main() {
	var prompt string
	if len(os.Args) > 1 {
		prompt = os.Args[1]
	}
	res, err := answer(
		prompt,
		os.Getenv("GIT_USERNAME"),
		os.Getenv("GIT_PASSWORD"),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(res)
}

// answer returns the response to a prompt from git. git runs the program named
// by GIT_ASKPASS with its prompt as the only argument and reads the response
// from stdout. A username prompt gets the username. Every other prompt gets the
// password.
func answer(prompt, username, password string) (string, error) {
	if strings.HasPrefix(prompt, "Username") {
		return username, nil
	}
	if password == "" {
		return "", errors.New("GIT_PASSWORD must be set")
	}
	return password, nil
}
