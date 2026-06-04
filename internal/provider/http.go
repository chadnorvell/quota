package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

var httpClient = &http.Client{Timeout: 18 * time.Second}

func get(ctx context.Context, url string, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "quota/0.1")
	req.Header.Set("Accept", "application/json,text/html;q=0.8,*/*;q=0.5")
	for key, value := range headers {
		if value != "" {
			req.Header.Set(key, value)
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if readErr != nil {
		return nil, resp.StatusCode, readErr
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return body, resp.StatusCode, fmt.Errorf("http %d", resp.StatusCode)
	}
	return body, resp.StatusCode, nil
}
