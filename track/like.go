package track

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/GuustTaillieu/Spotify-utilities/auth"
	"github.com/GuustTaillieu/Spotify-utilities/types"
)

func LikeCurrentTrack(authToken *auth.Token) error {
	track, err := GetCurrentTrack(authToken)
	if err != nil {
		return fmt.Errorf("error getting current track %v", err)
	}

	if err := LikeTrack(authToken, track.ID); err != nil {
		return fmt.Errorf("error saving current track %v", err)
	}
	return nil
}

// saveTrack saves (likes) the track using the Spotify API
func LikeTrack(authToken *auth.Token, trackID ID) error {
	payload := map[string][]string{"ids": {string(trackID)}}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal save track payload: %v", err)
	}

	req, err := http.NewRequest("PUT", "https://api.spotify.com/v1/me/tracks", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create save track request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+authToken.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to save track: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		var spotifyResp types.SpotifyResponse
		if err := json.Unmarshal(body, &spotifyResp); err == nil && spotifyResp.Error.Message != "" {
			return fmt.Errorf("failed to save track: %s (status: %d)", spotifyResp.Error.Message, spotifyResp.Error.Status)
		}
		return fmt.Errorf("failed to save track: status code %d", resp.StatusCode)
	}

	fmt.Println("Successfully liked the track!")
	return nil
}
