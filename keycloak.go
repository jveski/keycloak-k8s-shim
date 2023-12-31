package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Keycloak struct {
	client                             *http.Client
	url, realm, username, passwordPath string

	tokenLock      sync.Mutex
	accessToken    string
	accessTokenExp time.Time
}

func NewKeycloak(url, realm, username, passwordPath string, timeout time.Duration) (*Keycloak, error) {
	if url == "" {
		return nil, errors.New("keycloak URL is required")
	}
	if username == "" {
		return nil, errors.New("keycloak username is required")
	}
	if _, err := os.Stat(passwordPath); err != nil {
		return nil, errors.New("keycloak password file does not exist")
	}
	return &Keycloak{
		url:          url,
		realm:        realm,
		username:     username,
		passwordPath: passwordPath,
		client: &http.Client{
			Timeout:   timeout,
			Transport: &http.Transport{},
		},
	}, nil
}

func (k *Keycloak) Fetch(ctx context.Context, clientName string) ([]byte, error) {
	start := time.Now()
	defer func() {
		log.Printf("finished fetching client secret for %s in %dms", clientName, time.Since(start).Milliseconds())
	}()

	clientID, err := k.findClientID(ctx, clientName)
	if err != nil {
		return nil, fmt.Errorf("resolving client ID: %w", err)
	}

	return k.getClientSecret(ctx, clientID)
}

func (k *Keycloak) getClientSecret(ctx context.Context, id string) ([]byte, error) {
	token, err := k.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/admin/realms/%s/clients/%s/client-secret", k.url, k.realm, id)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := k.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body := struct {
		Value string `json:"value"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return nil, err
	}
	if body.Value == "" {
		return nil, status.Error(codes.FailedPrecondition, "client secret is empty")
	}

	return []byte(body.Value), nil
}

// findClientID maps client names to client IDs.
// Often the client name is referred to as "clientID" because the name is actually used by most clients,
// but the internal UUID is what we actually need to get the secret.
func (k *Keycloak) findClientID(ctx context.Context, name string) (string, error) {
	token, err := k.getAccessToken(ctx)
	if err != nil {
		return "", err
	}

	q := url.Values{}
	q.Add("clientId", name)
	url := fmt.Sprintf("%s/admin/realms/%s/clients?%s", k.url, k.realm, q.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := k.do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body := []struct {
		ID string `json:"id"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return "", err
	}
	if len(body) == 0 {
		return "", status.Error(codes.FailedPrecondition, "clientID not found")
	}

	return body[0].ID, nil
}

func (k *Keycloak) getAccessToken(ctx context.Context) (string, error) {
	k.tokenLock.Lock()
	defer k.tokenLock.Unlock()

	if time.Now().After(k.accessTokenExp) {
		if err := k.refreshAccessTokenUnlocked(ctx); err != nil {
			return "", fmt.Errorf("refreshing access token: %w", err)
		}
	}

	return k.accessToken, nil
}

func (k *Keycloak) refreshAccessTokenUnlocked(ctx context.Context) error {
	passwordBytes, err := os.ReadFile(k.passwordPath)
	if err != nil {
		return err
	}
	password := strings.TrimSpace(string(passwordBytes))

	q := url.Values{}
	q.Add("grant_type", "password")
	q.Add("username", k.username)
	q.Add("password", string(password))
	q.Add("client_id", "admin-cli")
	url := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", k.url, k.realm)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBufferString(q.Encode()))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := k.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body := struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return err
	}

	k.accessToken = body.AccessToken
	k.accessTokenExp = time.Now().Add(time.Second * time.Duration(body.ExpiresIn-(body.ExpiresIn/4)))
	log.Printf("refreshed keycloak token (expires in %ds)", body.ExpiresIn)
	return nil
}

func (k *Keycloak) do(req *http.Request) (*http.Response, error) {
	resp, err := k.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("server error status %d: %s", resp.StatusCode, body)
	}
	return resp, nil
}
