package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/GuustTaillieu/Spotify-utilities/auth"
	"github.com/GuustTaillieu/Spotify-utilities/track"
)

type CliFunction func(*auth.Token) error

var cliFunctions = map[string]CliFunction{
	"like_current_track": track.LikeCurrentTrack,
	"get_current_track": func(token *auth.Token) error {
		track, err := track.GetCurrentTrack(token)
		if err != nil {
			return err
		}
		fmt.Printf("The current track is playing: \"%s\"\n", track.Name)
		return nil
	},
}

func GetAvailableCommands() string {
	ac := make([]string, 0, len(cliFunctions))
	for c := range cliFunctions {
		ac = append(ac, c)
	}
	return strings.Join(ac, "\n ")
}

func main() {
	token, err := auth.GetOrRefreshToken()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting token: %v\n", err)
		os.Exit(1)
	}

	args := os.Args[1:]
	if len(args) < 1 {
		fmt.Printf("No command given, try one of the following:\n %s\n", GetAvailableCommands())
		os.Exit(0)
	}
	command := args[0]

	cliFunc, ok := cliFunctions[command]
	if !ok {
		fmt.Printf("Command not found: \"%v\".\nTry one of the following:\n %s\n", command, GetAvailableCommands())
		os.Exit(0)
	}
	err = cliFunc(token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Something went wrong: %v", command)
		os.Exit(1)
	}
}
