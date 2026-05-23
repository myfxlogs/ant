package systemai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	discoverTimeout = 15 * time.Second
	maxPages        = 10
	maxBodyBytes    = 2 << 20 // 2 MiB
)

var (
	errBaseURLEmpty = errors.New("base_url is empty")
	errBaseURLBad   = errors.New("base url format invalid")
)

func validateBaseURL(s string) error {
	u, err := url.Parse(s)
	if err != nil {
		return errBaseURLBad
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errBaseURLBad
	}
	if u.Host == "" {
		return errBaseURLBad
	}
	return nil
}

func authHeader(req *http.Request, secret string) {
	if s := strings.TrimSpace(secret); s != "" {
		req.Header.Set("Authorization", "Bearer "+s)
	}
}

// fetchOpenAIModels iterates the OpenAI-compatible /models endpoint and returns
// deduplicated model IDs. Pagination follows the `has_more` + `next` cursor
// convention; absent cursors stop the loop after one request.
func fetchOpenAIModels(ctx context.Context, baseURL, secret string) ([]string, error) {
	seen := map[string]struct{}{}
	out := make([]string, 0, 32)
	cursor := ""

	for page := 0; page < maxPages; page++ {
		ids, next, more, err := fetchModelsPage(ctx, baseURL, secret, cursor)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
		if !more || next == "" || next == cursor {
			break
		}
		cursor = next
	}
	if len(out) == 0 {
		return nil, errors.New("no models returned by upstream /models")
	}
	return out, nil
}

func fetchModelsPage(ctx context.Context, baseURL, secret, after string) ([]string, string, bool, error) {
	endpoint := baseURL + "/models"
	if after != "" {
		v := url.Values{}
		v.Set("after", after)
		endpoint += "?" + v.Encode()
	}

	rctx, cancel := context.WithTimeout(ctx, discoverTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(rctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", false, err
	}
	authHeader(req, secret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) && ue.Timeout() {
			return nil, "", false, errors.New("upstream timeout while requesting /models")
		}
		return nil, "", false, fmt.Errorf("upstream unreachable: %w", err)
	}
	defer resp.Body.Close()

	if msg := classifyHTTPError(resp); msg != "" {
		return nil, "", false, errors.New(msg)
	}

	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		HasMore bool   `json:"has_more"`
		Next    string `json:"next"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBodyBytes)).Decode(&payload); err != nil {
		return nil, "", false, errors.New("invalid /models response: cannot parse json")
	}
	ids := make([]string, 0, len(payload.Data))
	last := ""
	for _, m := range payload.Data {
		id := strings.TrimSpace(m.ID)
		if id == "" {
			continue
		}
		ids = append(ids, id)
		last = id
	}
	if payload.Next != "" && payload.HasMore {
		return ids, payload.Next, true, nil
	}
	return ids, last, payload.HasMore, nil
}

// classifyHTTPError returns "" if the response is OK; otherwise a short
// canonical error message inspected later by FriendlyError.
func classifyHTTPError(resp *http.Response) string {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return ""
	}
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return "upstream unauthorized: check api key/secret"
	case http.StatusNotFound:
		return "upstream endpoint not found: base_url must be openai-compatible"
	case http.StatusForbidden, http.StatusTooManyRequests:
		body := readBodyTrimmed(resp.Body, 1024)
		low := strings.ToLower(body)
		switch {
		case strings.Contains(low, "freetieronly") || strings.Contains(low, "free tier") || strings.Contains(low, "free-tier"):
			return fmt.Sprintf("upstream free-tier exhausted: status %d", resp.StatusCode)
		case strings.Contains(low, "quota") || strings.Contains(low, "exhaust") || strings.Contains(low, "rate limit") || strings.Contains(low, "too many requests"):
			return fmt.Sprintf("upstream quota exhausted: status %d", resp.StatusCode)
		case resp.StatusCode == http.StatusForbidden:
			return "upstream unauthorized: check api key/secret"
		default:
			return fmt.Sprintf("status %d %s", resp.StatusCode, low)
		}
	}
	body := readBodyTrimmed(resp.Body, 512)
	if body == "" {
		return fmt.Sprintf("status %d", resp.StatusCode)
	}
	return fmt.Sprintf("upstream returned error: status %d %s", resp.StatusCode, body)
}

func readBodyTrimmed(rc io.ReadCloser, n int64) string {
	b, _ := io.ReadAll(io.LimitReader(rc, n))
	return strings.TrimSpace(string(b))
}

// fetchZhipuModels handles Zhipu GLM's two known list shapes:
//   - {"data":[{"id"|"model"|"model_id":"..."}]}
//   - {"data":{"list":[{"id"|"model"|"model_id":"..."}]}}
func fetchZhipuModels(ctx context.Context, baseURL, secret string) ([]string, error) {
	q := url.Values{}
	q.Set("limit", "1000")
	q.Set("page_size", "1000")
	endpoint := baseURL + "/models?" + q.Encode()

	rctx, cancel := context.WithTimeout(ctx, discoverTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(rctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	authHeader(req, secret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if msg := classifyHTTPError(resp); msg != "" {
		return nil, errors.New(msg)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))

	type item struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		ModelID string `json:"model_id"`
	}
	pickID := func(it item) string {
		for _, v := range []string{it.ID, it.Model, it.ModelID} {
			if s := strings.TrimSpace(v); s != "" {
				return s
			}
		}
		return ""
	}

	var flat struct {
		Data []item `json:"data"`
	}
	if err := json.Unmarshal(body, &flat); err == nil && len(flat.Data) > 0 {
		return dedupe(flat.Data, pickID), nil
	}

	var nested struct {
		Data struct {
			List []item `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &nested); err == nil && len(nested.Data.List) > 0 {
		return dedupe(nested.Data.List, pickID), nil
	}
	return nil, errors.New("invalid /models response: cannot parse json")
}

func dedupe[T any](in []T, key func(T) string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, x := range in {
		k := key(x)
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	return out
}
