package main

import (
	"fmt"
	"os"

	"github.com/GuustTaillieu/Spotify-utilities/auth"
	"github.com/GuustTaillieu/Spotify-utilities/cli"
)

func main() {
	token, err := auth.GetOrRefreshToken()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting token: %v\n", err)
		os.Exit(1)
	}

	args := os.Args[1:]
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "No command given, try one of the following:\n %s\n", cli.GetAvailableCommands())
		os.Exit(1)
	}
	command := args[0]

	cliFunc, ok := cli.CliFunctions[command]
	if !ok {
		fmt.Fprintf(os.Stderr, "Command not found: \"%v\".\nTry one of the following:\n %s\n", command, cli.GetAvailableCommands())
		os.Exit(1)
	}
	err = cliFunc(token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Something went wrong: %v", err)
		os.Exit(1)
	}

	os.Exit(0)
}
