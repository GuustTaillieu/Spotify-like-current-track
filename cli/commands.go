package cli

import (
	"strings"

	"github.com/GuustTaillieu/Spotify-utilities/auth"
)

type CliFunction func(*auth.Token) error

var CliFunctions = map[string]CliFunction{
	"like_current_track":     CommandLikeCurrentTrack,
	"get_current_track":      CommandGetCurrentTrack,
	"is_current_track_liked": CommandIsCurrentTrackLiked,
}

func GetAvailableCommands() string {
	ac := make([]string, 0, len(CliFunctions))
	for c := range CliFunctions {
		ac = append(ac, c)
	}
	return strings.Join(ac, "\n ")
}
