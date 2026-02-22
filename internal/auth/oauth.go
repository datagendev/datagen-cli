package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	// ClientID is the OAuth 2.0 client identifier for the CLI.
	// The CLI is a public client — no client_secret.
	ClientID = "datagen-cli"

	// DefaultWebBaseURL is the base URL of the DataGen web application.
	// Overridable via DATAGEN_WEB_BASE_URL.
	DefaultWebBaseURL = "https://datagen.dev"

	// DefaultServerBaseURL is the base URL of the DataGen API server.
	// Overridable via DATAGEN_SERVER_URL.
	DefaultServerBaseURL = "https://api.datagen.dev"
)

// PKCEParams holds the generated PKCE verifier, challenge, and CSRF state.
type PKCEParams struct {
	Verifier  string
	Challenge string
	State     string
}

// OAuthTokens holds the access and refresh tokens returned after a successful exchange.
type OAuthTokens struct {
	AccessToken  string
	RefreshToken string
}

// WebBaseURL returns the configured web application base URL.
func WebBaseURL() string {
	if u := os.Getenv("DATAGEN_WEB_BASE_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return DefaultWebBaseURL
}

// ServerBaseURL returns the configured API server base URL.
func ServerBaseURL() string {
	if u := os.Getenv("DATAGEN_SERVER_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return DefaultServerBaseURL
}

// GeneratePKCE generates a PKCE code verifier, its S256 challenge, and a CSRF state value.
// Verifier: 64 random bytes, base64url-encoded (RFC 7636 §4.1).
// Challenge: BASE64URL(SHA256(ASCII(verifier))).
// State: 16 random bytes, base64url-encoded.
func GeneratePKCE() (*PKCEParams, error) {
	verifierBytes := make([]byte, 64)
	if _, err := rand.Read(verifierBytes); err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}
	verifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])

	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	return &PKCEParams{
		Verifier:  verifier,
		Challenge: challenge,
		State:     state,
	}, nil
}

// BuildAuthorizeURL constructs the OAuth authorization URL.
func BuildAuthorizeURL(webBaseURL, redirectURI string, pkce *PKCEParams) string {
	params := url.Values{}
	params.Set("client_id", ClientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", "read:user deployment:read deployment:run")
	params.Set("state", pkce.State)
	params.Set("code_challenge", pkce.Challenge)
	params.Set("code_challenge_method", "S256")
	return webBaseURL + "/login/oauth/authorize?" + params.Encode()
}

// StartCallbackServer starts a local HTTP server on a random port to capture the OAuth callback.
// Returns the port, a channel that emits the auth code on success or "error:..." on failure,
// and a stop function to shut down the server.
func StartCallbackServer(expectedState string) (port int, codeCh <-chan string, stop func(), err error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, nil, nil, fmt.Errorf("failed to start local callback server: %w", err)
	}
	port = listener.Addr().(*net.TCPAddr).Port

	ch := make(chan string, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		if errParam := q.Get("error"); errParam != "" {
			msg := errParam
			if desc := q.Get("error_description"); desc != "" {
				msg += ": " + desc
			}
			ch <- "error:" + msg
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<html><body><h2>Login failed</h2><p>%s</p><p>You may close this tab.</p></body></html>`, errParam)
			return
		}

		if q.Get("state") != expectedState {
			ch <- "error:state mismatch — possible CSRF attack"
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}

		code := q.Get("code")
		if code == "" {
			ch <- "error:no authorization code in callback"
			http.Error(w, "Missing code", http.StatusBadRequest)
			return
		}

		ch <- code
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><h2>Login successful!</h2><p>You may close this tab and return to the terminal.</p></body></html>`)
	})

	srv := &http.Server{Handler: mux}
	go srv.Serve(listener) //nolint:errcheck
	stop = func() { srv.Shutdown(context.Background()) } //nolint:errcheck

	return port, ch, stop, nil
}

// ExchangeCode exchanges an authorization code for access and refresh tokens.
// serverBaseURL should point to the Wasp API server (not the frontend).
// Falls back to JSON parsing if the content-type differs.
func ExchangeCode(serverBaseURL, redirectURI, code string, pkce *PKCEParams) (*OAuthTokens, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("client_id", ClientID)
	form.Set("code_verifier", pkce.Verifier)

	resp, err := http.Post(
		serverBaseURL+"/api/oauth/access_token",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed (%d): %s", resp.StatusCode, string(body))
	}

	var accessToken, refreshToken string
	ct := resp.Header.Get("Content-Type")

	if strings.Contains(ct, "application/x-www-form-urlencoded") {
		parsed, parseErr := url.ParseQuery(string(body))
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse token response: %w", parseErr)
		}
		accessToken = parsed.Get("access_token")
		refreshToken = parsed.Get("refresh_token")
	} else {
		// Fallback: try JSON (e.g. if content-type header is missing or set to application/json)
		var j struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		}
		if jsonErr := json.Unmarshal(body, &j); jsonErr != nil {
			return nil, fmt.Errorf("failed to parse token response (content-type: %s): %w", ct, jsonErr)
		}
		accessToken = j.AccessToken
		refreshToken = j.RefreshToken
	}

	if accessToken == "" {
		return nil, fmt.Errorf("no access_token in response")
	}

	return &OAuthTokens{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

// FetchApiKey uses the OAuth access token to retrieve the user's API key
// from the server. This is needed because /apps/ endpoints authenticate
// via X-API-Key (hashed lookup), not OAuth tokens.
func FetchApiKey(serverBaseURL, accessToken string) (string, error) {
	req, err := http.NewRequest("GET", serverBaseURL+"/api/oauth/api-key", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("api-key request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read api-key response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("api-key request failed (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		ApiKey string `json:"api_key"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse api-key response: %w", err)
	}
	if result.ApiKey == "" {
		return "", fmt.Errorf("no api_key in response")
	}
	return result.ApiKey, nil
}
