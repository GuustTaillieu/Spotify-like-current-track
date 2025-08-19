package auth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// Spotify API credentials (replace with your own)
const (
	clientFile  = ".custom_scripts/spotify/.client_credentials.json"
	redirectURI = "http://127.0.0.1:3000"
	scopes      = "user-read-currently-playing user-library-modify user-library-read"
	tokenFile   = ".custom_scripts/spotify/.spotify_tokens.json"
)

type clientCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// Token represents the structure for Spotify API tokens
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

// getOrRefreshToken retrieves or refreshes the access token
func GetOrRefreshToken() (*Token, error) {
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

func getClientSecrets() (clientCredentials, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return clientCredentials{}, fmt.Errorf("failed to get home directory: %v", err)
	}
	credentialsFile := filepath.Join(homeDir, clientFile)
	file, err := os.Open(credentialsFile)
	if err != nil {
		return clientCredentials{}, fmt.Errorf("failed to open credentials file: %v", err)
	}
	defer file.Close()

	var credentials clientCredentials
	body, err := io.ReadAll(file)
	if err != nil {
		return clientCredentials{}, fmt.Errorf("failed to read from file: %v", err)
	}
	if err := json.Unmarshal(body, &credentials); err != nil {
		return clientCredentials{}, fmt.Errorf("failed to parse credentials: %v", err)
	}

	return credentials, nil
}

// getAuthCode automates the authorization process by starting a local server and opening the browser
func getAuthCode() (string, error) {
	codeChan := make(chan string)
	errChan := make(chan error)

	// Start local HTTP server
	server := &http.Server{Addr: ":3000"}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		code := query.Get("code")
		if code == "" {
			errChan <- fmt.Errorf("no code found in redirect URL")
			return
		}
		codeChan <- code
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
			<!DOCTYPE html>
			<html>
			<body>
				<p>Authorization successful! This tab will close automatically.</p>
				<script>window.close();</script>
			</body>
			</html>
		`)
	})

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("server error: %v", err)
		}
	}()

	creds, err := getClientSecrets()
	if err != nil {
		return "", fmt.Errorf("Could not retrieve credentials: %v", err)
	}
	// Open authorization URL in default browser
	authURL := fmt.Sprintf(
		"https://accounts.spotify.com/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=%s",
		url.QueryEscape(creds.ClientID), url.QueryEscape(redirectURI), url.QueryEscape(scopes),
	)
	if err := openBrowser(authURL); err != nil {
		return "", fmt.Errorf("failed to open browser: %v", err)
	}

	// Wait for code or error
	select {
	case code := <-codeChan:
		if err := server.Close(); err != nil {
			fmt.Printf("Warning: failed to close server: %v\n", err)
		}
		return code, nil
	case err := <-errChan:
		if closeErr := server.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close server: %v\n", closeErr)
		}
		return "", err
	case <-time.After(5 * time.Minute): // Timeout after 5 minutes
		server.Close()
		return "", fmt.Errorf("authorization timeout")
	}
}

// openBrowser opens the URL in the default browser based on OS
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return exec.Command(cmd, args...).Start()
}

// getInitialTokens requests initial access and refresh tokens
func getInitialTokens(code string) (*Token, error) {
	creds, err := getClientSecrets()
	if err != nil {
		return nil, fmt.Errorf("Could not retrieve credentials: %v", err)
	}

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", creds.ClientID)
	data.Set("client_secret", creds.ClientSecret)

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

	creds, err := getClientSecrets()
	if err != nil {
		return nil, fmt.Errorf("Could not retrieve credentials: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(creds.ClientID+":"+creds.ClientSecret)))

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
