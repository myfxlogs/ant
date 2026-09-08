package systemai

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
)

// CurlImport holds provider config fields extracted from a vendor curl example.
type CurlImport struct {
	BaseURL      string // API root, endpoint suffix (/chat/completions, /models) stripped
	APIKey       string // empty when absent or a {your_key}-style placeholder
	DefaultModel string // "model" from the JSON body
	Models       []string
	NameHint     string   // display-name hint derived from the URL host
	Warnings     []string // human-readable notes for the user to review
}

var errCurlNoURL = errors.New("未能识别 URL：请粘贴完整的 curl 命令（含厂商文档中的请求地址）")

// ParseProviderCurlRaw extracts base_url / API key / model from a pasted curl
// example (POSIX-style quoting, `\` line continuations). Parse only — the
// caller decides what to apply; nothing is validated or persisted here beyond
// URL shape.
func ParseProviderCurlRaw(raw string) (*CurlImport, error) {
	toks := tokenizeCurl(lineContinuations(raw))
	res := &CurlImport{}
	u := ""
	body := ""
	for i := 0; i < len(toks); i++ {
		t := toks[i]
		switch {
		case t == "--url" && i+1 < len(toks):
			i++
			u = toks[i]
		case (t == "-H" || t == "--header") && i+1 < len(toks):
			i++
			name, val, _ := strings.Cut(toks[i], ":")
			applyCurlHeader(res, strings.ToLower(strings.TrimSpace(name)), strings.TrimSpace(val))
		case (t == "-d" || t == "--data" || t == "--data-raw" || t == "--data-binary" ||
			t == "--data-urlencode" || t == "--json") && i+1 < len(toks):
			i++
			body = toks[i]
		case t == "-X" || t == "--request" || t == "-u" || t == "--user" || t == "-A" || t == "--user-agent":
			i++ // known value-flag we don't use — skip its value
		case strings.HasPrefix(t, "http://") || strings.HasPrefix(t, "https://"):
			if u == "" {
				u = t
			}
		}
	}
	if u == "" {
		return nil, errCurlNoURL
	}
	res.BaseURL = normalizeAPIBase(strings.TrimRight(u, "/"))
	res.NameHint = curlNameHint(u)

	parseCurlBody(res, body)

	if res.APIKey != "" && isPlaceholderKey(res.APIKey) {
		res.APIKey = ""
		res.Warnings = append(res.Warnings, "示例中的 API Key 是占位符——请替换为你自己的真实 Key")
	}
	if res.APIKey == "" {
		res.Warnings = append(res.Warnings, "未识别到 API Key——请手动粘贴你的 Key")
	}
	return res, nil
}

func applyCurlHeader(res *CurlImport, name, val string) {
	if val == "" {
		return
	}
	switch name {
	case "authorization":
		if scheme, key, ok := strings.Cut(val, " "); ok && strings.EqualFold(strings.TrimSpace(scheme), "bearer") {
			res.APIKey = strings.TrimSpace(key)
		}
	case "x-api-key", "api-key":
		res.APIKey = val
	}
}

func parseCurlBody(res *CurlImport, body string) {
	body = strings.TrimSpace(body)
	if body == "" {
		res.Warnings = append(res.Warnings, "未在请求体中找到 model 字段——请手动填写模型名")
		return
	}
	var payload struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		res.Warnings = append(res.Warnings, "请求体不是合法 JSON，未能提取 model")
		return
	}
	if m := strings.TrimSpace(payload.Model); m != "" {
		res.DefaultModel = m
		res.Models = []string{m}
	} else {
		res.Warnings = append(res.Warnings, "未在请求体中找到 model 字段——请手动填写模型名")
	}
}

// isPlaceholderKey detects documentation placeholders like {your_key},
// <token>, $API_KEY — never treat them as a real credential.
func isPlaceholderKey(k string) bool {
	if len(k) > 120 {
		return false
	}
	low := strings.ToLower(k)
	return strings.Contains(low, "your") ||
		strings.Contains(k, "{") || strings.Contains(k, "<") ||
		strings.HasPrefix(k, "$")
}

// curlNameHint derives a short provider name from the URL host by skipping
// common infrastructure labels (token.sensenova.cn → sensanova,
// api.deepseek.com → deepseek).
func curlNameHint(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return ""
	}
	for _, l := range strings.Split(parsed.Hostname(), ".") {
		switch strings.ToLower(l) {
		case "api", "token", "open", "www", "chat", "gateway", "dashscope-compatible":
			continue
		}
		return l
	}
	return ""
}

// lineContinuations joins POSIX `\`-terminated lines so quoting stays intact.
func lineContinuations(s string) string {
	s = strings.ReplaceAll(s, "\\\r\n", " ")
	return strings.ReplaceAll(s, "\\\n", " ")
}

// tokenizeCurl splits a shell-style command into tokens, honoring single
// quotes (literal) and double quotes (with \", \\, \$ escapes). An
// unterminated quote ends the token gracefully — fail-open parsing.
func tokenizeCurl(s string) []string {
	var toks []string
	var cur strings.Builder
	inToken := false
	flush := func() {
		if inToken {
			toks = append(toks, cur.String())
			cur.Reset()
			inToken = false
		}
	}
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case c == '\'':
			inToken = true
			if j := strings.IndexByte(s[i+1:], '\''); j >= 0 {
				cur.WriteString(s[i+1 : i+1+j])
				i += j + 2
			} else {
				cur.WriteString(s[i+1:])
				i = len(s)
			}
		case c == '"':
			inToken = true
			i++
			for i < len(s) && s[i] != '"' {
				if s[i] == '\\' && i+1 < len(s) {
					cur.WriteByte(s[i+1])
					i += 2
				} else {
					cur.WriteByte(s[i])
					i++
				}
			}
			i++ // closing quote
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			flush()
			i++
		case c == '\\' && i+1 < len(s) && s[i+1] == '\n':
			i += 2 // safety: continuation outside quotes (normally pre-joined)
		default:
			inToken = true
			cur.WriteByte(c)
			i++
		}
	}
	flush()
	return toks
}
