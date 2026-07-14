package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Codex struct {
	credentialsDir string
	useFallbacks   bool
}

func NewCodex() Codex {
	return Codex{credentialsDir: defaultCredentialsDir(".codex"), useFallbacks: true}
}

// NewCodexFromDir creates an additional Codex account backed by dir/auth.json.
func NewCodexFromDir(dir string) Codex {
	return Codex{credentialsDir: expandHome(dir)}
}

func (c Codex) Name() string {
	return accountName("Codex", c.credentialsDir, ".codex")
}

func (c Codex) Fetch(ctx context.Context) Snapshot {
	auth := readCodexAuth(c.credentialsDir)
	if auth.AccessToken != "" {
		if snapshot, err := fetchCodexOAuth(ctx, auth); err == nil {
			snapshot.Name = c.Name()
			return snapshot
		}
	}

	cookie := ""
	if c.useFallbacks {
		cookie = firstEnv("QUOTA_CODEX_COOKIE", "OPENAI_COOKIE", "CHATGPT_COOKIE")
	}
	if cookie != "" {
		snapshot := fetchCodexDashboard(ctx, cookie)
		snapshot.Name = c.Name()
		return snapshot
	}

	if auth.Account != "" || auth.HasToken {
		status := "signed in"
		if !auth.HasToken {
			status = "account found"
		}
		return Snapshot{
			Name:      c.Name(),
			Account:   auth.Account,
			Source:    auth.Path,
			Status:    status,
			Notes:     []string{"Set QUOTA_CODEX_COOKIE to fetch chatgpt.com/codex/settings/usage."},
			UpdatedAt: time.Now(),
		}
	}

	return NewUnavailable(c.Name(), "local auth", "no readable "+filepath.Join(c.credentialsDir, "auth.json"))
}

func fetchCodexOAuth(ctx context.Context, auth codexAuth) (Snapshot, error) {
	url := resolveCodexUsageURL()
	headers := map[string]string{
		"Authorization": "Bearer " + auth.AccessToken,
		"User-Agent":    "CodexBar",
		"Accept":        "application/json",
	}
	if auth.AccountID != "" {
		headers["ChatGPT-Account-Id"] = auth.AccountID
	}
	body, _, err := get(ctx, url, headers)
	if err != nil {
		return Snapshot{}, err
	}
	value, ok := decodeAny(body)
	if !ok {
		return Snapshot{}, fmt.Errorf("response was not JSON")
	}
	lanes := codexOAuthLanes(value)
	plan, _ := stringAt(value, "plan_type")
	return Snapshot{
		Name:      "Codex",
		Plan:      plan,
		Account:   auth.Account,
		Source:    "codex-oauth",
		Status:    "ok",
		Lanes:     lanes,
		UpdatedAt: time.Now(),
	}, nil
}

func codexOAuthLanes(value any) []Lane {
	var lanes []Lane
	addWindow := func(label string, paths ...[]string) {
		for _, path := range paths {
			if window, ok := windowAt(value, path...); ok {
				lanes = append(lanes, windowLane(label, window))
				return
			}
		}
	}
	addWindow("5h limit", []string{"rate_limit", "primary_window"}, []string{"primary_window"})
	addWindow("weekly limit", []string{"rate_limit", "secondary_window"}, []string{"secondary_window"})

	if root, ok := value.(map[string]any); ok {
		if credits, ok := root["credits"].(map[string]any); ok {
			hasCredits := false
			if value, ok := credits["has_credits"].(bool); ok {
				hasCredits = value
			}
			if balance, ok := number(credits["balance"]); ok && hasCredits {
				lanes = append(lanes, Lane{
					Label:  "Credits",
					Used:   balance,
					Detail: fmt.Sprintf("%.2f remaining", balance),
				})
			}
		}

		if extras, ok := root["additional_rate_limits"].([]any); ok {
			for _, extra := range extras {
				object, ok := extra.(map[string]any)
				if !ok {
					continue
				}
				label, _ := object["limit_name"].(string)
				if label == "" {
					label, _ = object["metered_feature"].(string)
				}
				if label == "" {
					label = "Extra limit"
				}
				if window, ok := windowAt(object, "rate_limit", "primary_window"); ok {
					lanes = append(lanes, windowLane(label+" 5h", window))
				}
				if window, ok := windowAt(object, "rate_limit", "secondary_window"); ok {
					lanes = append(lanes, windowLane(label+" weekly", window))
				}
			}
		}
	}
	return lanes
}

func windowAt(value any, path ...string) (map[string]any, bool) {
	current := value
	for _, key := range path {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[key]
		if !ok {
			return nil, false
		}
	}
	window, ok := current.(map[string]any)
	return window, ok
}

func windowLane(label string, window map[string]any) Lane {
	used, _ := number(window["used_percent"])
	reset := ""
	if resetAt, ok := number(window["reset_at"]); ok && resetAt > 0 {
		reset = time.Unix(int64(resetAt), 0).Format("Jan 2 15:04")
	}
	detail := fmt.Sprintf("%.0f%% used", used)
	if reset != "" {
		detail += " reset " + reset
	}
	return Lane{
		Label:   label,
		Used:    used,
		Limit:   100,
		Percent: percent(used, 100),
		Unit:    "%",
		Detail:  detail,
		Reset:   reset,
	}
}

func fetchCodexDashboard(ctx context.Context, cookie string) Snapshot {
	body, _, err := get(ctx, "https://chatgpt.com/codex/settings/usage", map[string]string{
		"Cookie": cookie,
	})
	if err != nil {
		return NewUnavailable("Codex", "openai-web", err.Error())
	}

	lanes := htmlPercentLanes(body, "Session", "Weekly", "Monthly", "Messages", "Codex")
	if value, ok := decodeAny(body); ok {
		lanes = append(lanes, codexJSONLanes(value)...)
	}
	if len(lanes) == 0 {
		lanes = append(lanes, Lane{Label: "Dashboard", Detail: fmt.Sprintf("fetched %d bytes; parser needs response shape sample", len(body))})
	}

	return Snapshot{
		Name:      "Codex",
		Source:    "openai-web",
		Status:    "ok",
		Lanes:     dedupeLanes(lanes),
		UpdatedAt: time.Now(),
	}
}

func codexJSONLanes(value any) []Lane {
	var lanes []Lane
	for _, pair := range [][3]string{
		{"Session", "session_used", "session_limit"},
		{"Weekly", "weekly_used", "weekly_limit"},
		{"Monthly", "monthly_used", "monthly_limit"},
	} {
		used, usedOK := firstNumber(value, pair[1], pair[1]+"_messages")
		limit, limitOK := firstNumber(value, pair[2], pair[2]+"_messages")
		if usedOK && limitOK {
			lanes = append(lanes, Lane{
				Label:   pair[0],
				Used:    used,
				Limit:   limit,
				Percent: percent(used, limit),
				Detail:  fmt.Sprintf("%.0f / %.0f", used, limit),
			})
		}
	}
	return lanes
}

type codexAuth struct {
	Path        string
	Account     string
	AccountID   string
	AccessToken string
	HasToken    bool
}

func readCodexAuth(credentialsDir string) codexAuth {
	path := filepath.Join(credentialsDir, "auth.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return codexAuth{}
	}
	var raw map[string]any
	_ = json.Unmarshal(data, &raw)
	account := ""
	for _, keys := range [][]string{
		{"account", "email"},
		{"user", "email"},
		{"profile", "email"},
	} {
		if text, ok := stringAt(raw, keys...); ok {
			account = text
			break
		}
	}
	accessToken := ""
	accountID := ""
	if tokens, ok := raw["tokens"].(map[string]any); ok {
		accessToken, _ = stringAt(tokens, "access_token")
		if accessToken == "" {
			accessToken, _ = stringAt(tokens, "accessToken")
		}
		accountID, _ = stringAt(tokens, "account_id")
		if accountID == "" {
			accountID, _ = stringAt(tokens, "accountId")
		}
	}
	_, access := firstNumber(raw, "expires_at", "expiresAt")
	return codexAuth{
		Path:        path,
		Account:     account,
		AccountID:   accountID,
		AccessToken: accessToken,
		HasToken:    access || accessToken != "" || containsKey(raw, "tokens") || containsKey(raw, "access_token"),
	}
}

func resolveCodexUsageURL() string {
	base := "https://chatgpt.com/backend-api"
	if value := os.Getenv("CHATGPT_BASE_URL"); value != "" {
		base = value
	}
	if len(base) > 0 && base[len(base)-1] == '/' {
		base = base[:len(base)-1]
	}
	if base == "https://chatgpt.com" || base == "https://chat.openai.com" {
		base += "/backend-api"
	}
	if strings.Contains(base, "/backend-api") {
		return base + "/wham/usage"
	}
	return base + "/api/codex/usage"
}

func containsKey(value any, key string) bool {
	found := false
	walk(value, func(candidate string, _ any) {
		if normalizeKey(candidate) == normalizeKey(key) {
			found = true
		}
	})
	return found
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}

func dedupeLanes(lanes []Lane) []Lane {
	seen := map[string]bool{}
	var result []Lane
	for _, lane := range lanes {
		if seen[lane.Label] {
			continue
		}
		seen[lane.Label] = true
		result = append(result, lane)
	}
	return result
}
