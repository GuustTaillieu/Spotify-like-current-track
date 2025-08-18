package types

// SpotifyResponse represents common Spotify API response structures
type SpotifyResponse struct {
	Error struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	} `json:"error"`
}
