package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"
)

type KeycloakConfig struct {
	Credentials Credentials `json:"credentials"`
	TokenURL    string      `json:"token_url"`
}

type Credentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type KeycloakToken struct {
	Raw        string    `json:"raw"`
	Expiration time.Time `json:"expiration"`
}
type KeycloakTokenManager struct {
	Config     KeycloakConfig
	token      *KeycloakToken
	httpClient *http.Client
}

func NewKeycloakTokenManager(config KeycloakConfig) *KeycloakTokenManager {
	return &KeycloakTokenManager{
		Config: config,
	}
}

func (token *KeycloakToken) IsValid() bool {
	if token == nil {
		return false
	}
	return token.Expiration.After(time.Now())
}

func (manager *KeycloakTokenManager) GetValidToken() (string, error) {
	if manager.token.IsValid() {
		return manager.token.Raw, nil
	}

	return manager.fetchAndStoreToken()
}

func (manager *KeycloakTokenManager) fetchAndStoreToken() (string, error) {
	form := url.Values{}
	form.Add("grant_type", "client_credentials")
	form.Add("client_id", manager.Config.Credentials.ClientID)
	form.Add("client_secret", manager.Config.Credentials.ClientSecret)

	tokenURL := manager.Config.TokenURL

	req, err := http.NewRequest("POST", tokenURL, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := manager.httpClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", errors.New("failed to get token: " + string(body))
	}

	var responseData struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"` // seconds
	}

	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return "", err
	}

	manager.token = &KeycloakToken{
		Raw:        responseData.AccessToken,
		Expiration: time.Now().Add(time.Duration(responseData.ExpiresIn) * time.Second),
	}

	return responseData.AccessToken, nil
}
