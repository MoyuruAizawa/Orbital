package github

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const githubAPIBaseURL = "https://api.github.com"

type githubAppAccessTokenResponse struct {
	Token string `json:"token"`
}

type githubRunnerRegistrationTokenResponse struct {
	Token string `json:"token"`
}

type RunnerRegistrationToken struct {
	Token string
}

func GenerateRunnerRegistrationToken(ctx context.Context, org string, appID, installationID int64, pemPath string) (string, error) {
	if strings.TrimSpace(org) == "" {
		return "", fmt.Errorf("org is required")
	}
	if appID <= 0 {
		return "", fmt.Errorf("appID must be greater than 0")
	}
	if installationID <= 0 {
		return "", fmt.Errorf("installationID must be greater than 0")
	}
	if strings.TrimSpace(pemPath) == "" {
		return "", fmt.Errorf("pemPath is required")
	}

	privateKey, err := loadRSAPrivateKey(pemPath)
	if err != nil {
		return "", fmt.Errorf("load private key: %w", err)
	}

	jwtToken, err := createGitHubAppJWT(appID, privateKey, time.Now())
	if err != nil {
		return "", fmt.Errorf("create github app jwt: %w", err)
	}

	installationToken, err := requestInstallationAccessToken(ctx, jwtToken, installationID)
	if err != nil {
		return "", fmt.Errorf("request installation access token: %w", err)
	}

	registrationToken, err := requestRunnerRegistrationToken(ctx, installationToken, org)
	if err != nil {
		return "", fmt.Errorf("request runner registration token: %w", err)
	}

	return registrationToken, nil
}

func loadRSAPrivateKey(pemPath string) (*rsa.PrivateKey, error) {
	expandedPath, err := expandHome(pemPath)
	if err != nil {
		return nil, fmt.Errorf("expand home dir: %w", err)
	}

	absPath, err := filepath.Abs(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("resolve pem path: %w", err)
	}

	pemBytes, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read pem file %q: %w", absPath, err)
	}

	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("invalid pem file %q", absPath)
	}

	if privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return privateKey, nil
	}

	parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	privateKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not rsa")
	}

	return privateKey, nil
}

func expandHome(path string) (string, error) {
	if path == "~" {
		return os.UserHomeDir()
	}
	if strings.HasPrefix(path, "~") {
		home, error := os.UserHomeDir()
		if error != nil {
			return "", error
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

func createGitHubAppJWT(appID int64, privateKey *rsa.PrivateKey, now time.Time) (string, error) {
	headerJSON, err := json.Marshal(map[string]string{
		"alg": "RS256",
		"typ": "JWT",
	})
	if err != nil {
		return "", fmt.Errorf("marshal jwt header: %w", err)
	}

	payloadJSON, err := json.Marshal(map[string]any{
		"iat": now.Add(-time.Minute).Unix(),
		"exp": now.Add(9 * time.Minute).Unix(),
		"iss": strconv.FormatInt(appID, 10),
	})
	if err != nil {
		return "", fmt.Errorf("marshal jwt payload: %w", err)
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := encodedHeader + "." + encodedPayload

	hash := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func requestInstallationAccessToken(ctx context.Context, jwtToken string, installationID int64) (string, error) {
	url := fmt.Sprintf("%s/app/installations/%d/access_tokens", githubAPIBaseURL, installationID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var tokenResp githubAppAccessTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if tokenResp.Token == "" {
		return "", fmt.Errorf("installation access token is empty")
	}

	return tokenResp.Token, nil
}

func requestRunnerRegistrationToken(ctx context.Context, installationToken, org string) (string, error) {
	url := fmt.Sprintf("%s/orgs/%s/actions/runners/registration-token", githubAPIBaseURL, org)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+installationToken)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var tokenResp githubRunnerRegistrationTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if tokenResp.Token == "" {
		return "", fmt.Errorf("registration token is empty")
	}

	return tokenResp.Token, nil
}
