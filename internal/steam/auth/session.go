package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"
)

type Session struct {
	Client  *http.Client
	BaseURL string
	Origin  string
}

func NewSession() *Session {
	jar, _ := cookiejar.New(nil)
	return &Session{
		Client: &http.Client{
			Timeout: 20 * time.Second,
			Jar:     jar,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
		},
		BaseURL: "https://api.steampowered.com",
		Origin:  "https://steamcommunity.com",
	}
}

func (s *Session) Do(ctx context.Context, req *http.Request) ([]byte, []*http.Cookie, error) {
	req = req.WithContext(ctx)

	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "steam-http/0.1 (+golang)")
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.Cookies(), fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}

	return body, resp.Cookies(), nil
}
