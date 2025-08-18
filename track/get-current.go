package track

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/GuustTaillieu/Spotify-utilities/auth"
)

// ID for a Track
type ID string

// CurrentlyPlaying represents the structure for the currently playing track
type Track struct {
	ID   ID     `json:"id"`
	Name string `json:"name"`
}

type CurrentlyPlaying struct {
	Track     `json:"item"`
	IsPlaying bool `json:"is_playing"`
}

// GetCurrentTrack fetches the currently playing track ID
func GetCurrentTrack(authToken *auth.Token) (Track, error) {
	req, err := http.NewRequest("GET", "https://api.spotify.com/v1/me/player/currently-playing", nil)
	if err != nil {
		return Track{}, fmt.Errorf("failed to create current track request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+authToken.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return Track{}, fmt.Errorf("failed to fetch current track: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		return Track{}, fmt.Errorf("no content: no track is currently playing")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Track{}, fmt.Errorf("failed to read current track response: %v", err)
	}

	var currentPlaying CurrentlyPlaying
	if err := json.Unmarshal(body, &currentPlaying); err != nil {
		return Track{}, fmt.Errorf("failed to parse current track response: %v", err)
	}

	if currentPlaying.Track.ID == "" || !currentPlaying.IsPlaying {
		return Track{}, fmt.Errorf("no valid track ID found or no track is playing")
	}

	return currentPlaying.Track, nil
}
