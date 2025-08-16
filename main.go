package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// Spotify API credentials (replace with your own)
const (
	clientID     = "5d1d59669ef940fa9dc772d93ff64609"
	clientSecret = "838e21b6a3084acab7bc4a5175e589c2"
	redirectURI  = "http://127.0.0.1:3000"
	scopes       = "user-read-currently-playing user-library-modify"
	tokenFile    = ".custom_scripts/spotify/.spotify_tokens.json"
)

// Token represents the structure for Spotify API tokens
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

// SpotifyResponse represents common Spotify API response structures
type SpotifyResponse struct {
	Error struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	} `json:"error"`
}

// CurrentlyPlaying represents the structure for the currently playing track
type CurrentlyPlaying struct {
	Item struct {
		ID  string `json:"id"`
		URI string `json:"uri"`
	} `json:"item"`
	IsPlaying bool `json:"is_playing"`
}

func main() {
	fmt.Println("Starting Spotify Like Track program...")

	// Step 1: Get or refresh token
	token, err := getOrRefreshToken()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting token: %v\n", err)
		os.Exit(1)
	}

	// Step 2: Get current track ID
	trackID, err := getCurrentTrackID(token.AccessToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current track: %v\n", err)
		os.Exit(1)
	}

	// Step 3: Save (like) the track
	if err := saveTrack(token.AccessToken, trackID); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving track: %v\n", err)
		os.Exit(1)
	}
}

// getAuthCode prompts the user to get an authorization code
func getAuthCode() (string, error) {
	authURL := fmt.Sprintf(
		"https://accounts.spotify.com/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=%s",
		url.QueryEscape(clientID), url.QueryEscape(redirectURI), url.QueryEscape(scopes),
	)
	fmt.Printf("Open this URL in your browser to authorize:\n%s\n", authURL)
	fmt.Println("After authorizing, copy the 'code' parameter from the redirect URL.")
	fmt.Print("Enter the authorization code: ")
	var code string
	fmt.Scanln(&code)
	if code == "" {
		return "", fmt.Errorf("no authorization code provided")
	}
	return code, nil
}

// getInitialTokens requests initial access and refresh tokens
func getInitialTokens(code string) (*Token, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)

	req, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request tokens: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %v", err)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Error        struct {
			Status  int    `json:"status"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %v", err)
	}
	if tokenResp.Error.Message != "" {
		return nil, fmt.Errorf("token request failed: %s (status: %d)", tokenResp.Error.Message, tokenResp.Error.Status)
	}

	return &Token{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Unix() + int64(tokenResp.ExpiresIn) - 60,
	}, nil
}

// refreshTokens refreshes the access token using the refresh token
func refreshTokens(refreshToken string) (*Token, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh token request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(clientID+":"+clientSecret)))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read refresh token response: %v", err)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Error        struct {
			Status  int    `json:"status"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse refresh token response: %v", err)
	}
	if tokenResp.Error.Message != "" {
		return nil, fmt.Errorf("refresh token failed: %s (status: %d)", tokenResp.Error.Message, tokenResp.Error.Status)
	}

	newRefreshToken := refreshToken
	if tokenResp.RefreshToken != "" {
		newRefreshToken = tokenResp.RefreshToken
	}

	return &Token{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    time.Now().Unix() + int64(tokenResp.ExpiresIn) - 60,
	}, nil
}

// saveTokens saves tokens to a file
func saveTokens(token *Token) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}
	tokenPath := filepath.Join(homeDir, tokenFile)
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tokens: %v", err)
	}
	if err := os.WriteFile(tokenPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write tokens to file: %v", err)
	}
	return nil
}

// loadTokens loads tokens from a file
func loadTokens() (*Token, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}
	tokenPath := filepath.Join(homeDir, tokenFile)
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read token file: %v", err)
	}
	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token file: %v", err)
	}
	return &token, nil
}

// getOrRefreshToken retrieves or refreshes the access token
func getOrRefreshToken() (*Token, error) {
	token, err := loadTokens()
	if err != nil || token.AccessToken == "" || token.RefreshToken == "" {
		fmt.Println("No valid tokens found, initiating authorization...")
		code, err := getAuthCode()
		if err != nil {
			return nil, err
		}
		token, err = getInitialTokens(code)
		if err != nil {
			return nil, err
		}
		if err := saveTokens(token); err != nil {
			return nil, err
		}
		return token, nil
	}

	if time.Now().Unix() >= token.ExpiresAt {
		fmt.Println("Access token expired, refreshing...")
		newToken, err := refreshTokens(token.RefreshToken)
		if err != nil {
			fmt.Println("Refresh failed, re-authorizing...")
			code, err := getAuthCode()
			if err != nil {
				return nil, err
			}
			newToken, err = getInitialTokens(code)
			if err != nil {
				return nil, err
			}
		}
		if err := saveTokens(newToken); err != nil {
			return nil, err
		}
		return newToken, nil
	}

	return token, nil
}

// getCurrentTrackID fetches the currently playing track ID
func getCurrentTrackID(accessToken string) (string, error) {
	req, err := http.NewRequest("GET", "https://api.spotify.com/v1/me/player/currently-playing", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create current track request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch current track: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		return "", fmt.Errorf("no content: no track is currently playing")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read current track response: %v", err)
	}

	var currentPlaying CurrentlyPlaying
	if err := json.Unmarshal(body, &currentPlaying); err != nil {
		return "", fmt.Errorf("failed to parse current track response: %v", err)
	}

	if currentPlaying.Item.ID == "" || !currentPlaying.IsPlaying {
		return "", fmt.Errorf("no valid track ID found or no track is playing")
	}

	return currentPlaying.Item.ID, nil
}

// saveTrack saves (likes) the track using the Spotify API
func saveTrack(accessToken, trackID string) error {
	payload := map[string][]string{"ids": {trackID}}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal save track payload: %v", err)
	}

	req, err := http.NewRequest("PUT", "https://api.spotify.com/v1/me/tracks", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create save track request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to save track: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		var spotifyResp SpotifyResponse
		if err := json.Unmarshal(body, &spotifyResp); err == nil && spotifyResp.Error.Message != "" {
			return fmt.Errorf("failed to save track: %s (status: %d)", spotifyResp.Error.Message, spotifyResp.Error.Status)
		}
		return fmt.Errorf("failed to save track: status code %d", resp.StatusCode)
	}

	fmt.Println("Successfully liked the track!")
	return nil
}
