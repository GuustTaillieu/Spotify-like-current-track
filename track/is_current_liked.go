package track

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/GuustTaillieu/Spotify-utilities/auth"
)

func IsCurrentTrackLiked(authToken *auth.Token) (bool, error) {
	track, err := GetCurrentTrack(authToken)
	if err != nil {
		return false, fmt.Errorf("Something went wrong while getting the current track: %v", err)
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.spotify.com/v1/me/tracks/contains?ids=%s", track.ID), nil)
	if err != nil {
		return false, fmt.Errorf("failed to create current track request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+authToken.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to check current track: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read current track response: %v", err)
	}

	var tracksLiked []bool
	if err := json.Unmarshal(body, &tracksLiked); err != nil {
		return false, fmt.Errorf("failed to parse current track response: %v", err)
	}
	if len(tracksLiked) < 1 {
		return false, fmt.Errorf("failed to receive results from saved tracks: %v", err)
	}
	return tracksLiked[0], nil
}
