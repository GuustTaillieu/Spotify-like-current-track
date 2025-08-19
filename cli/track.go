package cli

import (
	"fmt"

	"github.com/GuustTaillieu/Spotify-utilities/auth"
	"github.com/GuustTaillieu/Spotify-utilities/track"
)

func CommandLikeCurrentTrack(token *auth.Token) error {
	err := track.LikeCurrentTrack(token)
	if err != nil {
		return err
	}
	fmt.Printf("Liked current track!")
	return nil
}

func CommandGetCurrentTrack(token *auth.Token) error {
	track, err := track.GetCurrentTrack(token)
	if err != nil {
		return err
	}
	fmt.Printf("The current track is playing: \"%s\"\n", track.Name)
	return nil
}

func CommandIsCurrentTrackLiked(token *auth.Token) error {
	isLiked, err := track.IsCurrentTrackLiked(token)
	if err != nil {
		return err
	}
	if !isLiked {
		return fmt.Errorf("Could not like song")
	}
	fmt.Printf("Liked song!")
	return nil
}
