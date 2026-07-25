package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const LINE_VERIFT_URL = "https://api.line.me/oauth2/v2.1/verify"

// lineVerifier verifies LIFF ID tokens against LINE's own verify endpoint —
// https://developers.line.biz/en/reference/line-login/#verify-id-token.
// No JWKS/local JWT verification needed; LINE does it and hands back the
// decoded claims.
type lineVerifier struct {
	channelID string
	client    *http.Client
}

func newLineVerifier(channelID string) *lineVerifier {
	return &lineVerifier{channelID: channelID, client: &http.Client{Timeout: 5 * time.Second}}
}

func (v *lineVerifier) VerifyIDToken(ctx context.Context, idToken string) (string, error) {
	form := url.Values{"id_token": {idToken}, "client_id": {v.channelID}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, LINE_VERIFT_URL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("lineauth: build verify request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := v.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("lineauth: verify id token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("lineauth: verify id token: status %d", resp.StatusCode)
	}

	var body struct {
		Sub string `json:"sub"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("lineauth: decode verify response: %w", err)
	}
	if body.Sub == "" {
		return "", errors.New("lineauth: verify response missing sub")
	}
	return body.Sub, nil
}
