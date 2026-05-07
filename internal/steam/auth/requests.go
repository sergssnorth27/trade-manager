package auth

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
)

func (s *Session) GetSessionId(ctx context.Context) ([]byte, []*http.Cookie, error) {
	url := "https://steamcommunity.com/login/home"
	req, err := http.NewRequest("GET", url, nil)
	body, cookies, err := s.Do(ctx, req)
	return body, cookies, err
}

func (s *Session) GetPasswordRSAPublicKey(ctx context.Context, accountName string) ([]byte, []*http.Cookie, error) {
	inputB64, err := SerializeCAuthentication_GetPasswordRSAPublicKey_Request(accountName)
	if err != nil {
		return nil, nil, fmt.Errorf("serialize request: %w", err)
	}

	u, err := url.Parse(s.BaseURL + "/IAuthenticationService/GetPasswordRSAPublicKey/v1")
	if err != nil {
		return nil, nil, fmt.Errorf("parse url: %w", err)
	}

	q := u.Query()
	q.Set("origin", s.Origin)
	q.Set("input_protobuf_encoded", inputB64)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("build request: %w", err)
	}

	body, cookies, err := s.Do(ctx, req)
	if err != nil {
		return nil, nil, fmt.Errorf("do request: %w", err)
	}
	return body, cookies, nil
}

func (s *Session) BeginAuthSessionViaCredentials(
	ctx context.Context,
	accountName, encryptedPassword string,
	encryptionTimestamp uint64,
) ([]byte, []*http.Cookie, error) {

	inputB64, err := SerializeCAuthentication_BeginAuthSessionViaCredentials_Request(
		accountName,
		encryptedPassword,
		encryptionTimestamp,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("serialize request: %w", err)
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if err := w.WriteField("input_protobuf_encoded", inputB64); err != nil {
		return nil, nil, fmt.Errorf("write field: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, nil, fmt.Errorf("writer close: %w", err)
	}

	u := s.BaseURL + "/IAuthenticationService/BeginAuthSessionViaCredentials/v1"
	req, err := http.NewRequestWithContext(ctx, "POST", u, &buf)
	if err != nil {
		return nil, nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Origin", s.Origin)
	req.Header.Set("Accept", "application/json, text/plain, */*")

	body, cookies, err := s.Do(ctx, req)
	if err != nil {
		return nil, nil, fmt.Errorf("do request: %w", err)
	}
	return body, cookies, nil
}

func (s *Session) UpdateAuthSessionWithSteamGuardCode(
	ctx context.Context,
	сlientId, steamId uint64,
	code string,
) ([]byte, []*http.Cookie, error) {

	inputB64, err := SerializeCAuthentication_UpdateAuthSessionWithSteamGuardCode_Request(
		сlientId,
		steamId,
		code,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("serialize request: %w", err)
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if err := w.WriteField("input_protobuf_encoded", inputB64); err != nil {
		return nil, nil, fmt.Errorf("write field: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, nil, fmt.Errorf("writer close: %w", err)
	}

	u := s.BaseURL + "/IAuthenticationService/UpdateAuthSessionWithSteamGuardCode/v1"
	req, err := http.NewRequestWithContext(ctx, "POST", u, &buf)
	if err != nil {
		return nil, nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Origin", s.Origin)
	req.Header.Set("Accept", "application/json, text/plain, */*")

	body, cookies, err := s.Do(ctx, req)
	if err != nil {
		return nil, nil, fmt.Errorf("do request: %w", err)
	}
	return body, cookies, nil
}

func (s *Session) PollAuthSessionStatus(
	ctx context.Context,
	сlientId uint64, requestId []byte,
	tokenToRevoke uint64,
) ([]byte, []*http.Cookie, error) {

	inputB64, err := SerializePollAuthSessionStatus_Request(
		сlientId,
		requestId,
		tokenToRevoke,
	)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if err := w.WriteField("input_protobuf_encoded", inputB64); err != nil {
		return nil, nil, fmt.Errorf("write field: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, nil, fmt.Errorf("writer close: %w", err)
	}

	u := s.BaseURL + "/IAuthenticationService/PollAuthSessionStatus/v1"
	req, err := http.NewRequestWithContext(ctx, "POST", u, &buf)
	if err != nil {
		return nil, nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Origin", s.Origin)
	req.Header.Set("Accept", "application/json, text/plain, */*")

	body, cookies, err := s.Do(ctx, req)
	if err != nil {
		return nil, nil, fmt.Errorf("do request: %w", err)
	}
	return body, cookies, nil
}

func (s *Session) FinalizeLogin(
	ctx context.Context,
	accessToken, sessionID string) ([]byte, []*http.Cookie, error) {

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if err := w.WriteField("nonce", accessToken); err != nil {
		return nil, nil, fmt.Errorf("write field: %w", err)
	}
	if err := w.WriteField("sessionid", sessionID); err != nil {
		return nil, nil, fmt.Errorf("write field: %w", err)
	}
	if err := w.WriteField("redir", "https://steamcommunity.com/login/home/?goto="); err != nil {
		return nil, nil, fmt.Errorf("write field: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, nil, fmt.Errorf("write field: %w", err)
	}

	u := "https://login.steampowered.com/jwt/finalizelogin"
	req, err := http.NewRequestWithContext(ctx, "POST", u, &buf)
	if err != nil {
		return nil, nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Origin", s.Origin)
	req.Header.Set("Accept", "application/json, text/plain, */*")

	body, cookies, err := s.Do(ctx, req)
	if err != nil {
		return nil, nil, fmt.Errorf("do request: %w", err)
	}
	return body, cookies, nil
}

func (s *Session) SetCookiesAndAuthentication(
	ctx context.Context, steamLoginSecure string) ([]byte, []*http.Cookie, error) {

	u, err := url.Parse("https://steamcommunity.com/market/search?appid=730")
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Origin", s.Origin)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	s.Client.Jar.SetCookies(u, []*http.Cookie{
		{
			Name:  "steamLoginSecure",
			Value: steamLoginSecure,
			Path:  "/",
		},
	})

	body, cookies, err := s.Do(ctx, req)
	if err != nil {
		return nil, nil, fmt.Errorf("do request: %w", err)
	}
	return body, cookies, nil
}
