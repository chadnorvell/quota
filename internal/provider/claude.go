package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Claude struct {
	credentialsDir string
	useFallbacks   bool
}

func NewClaude() Claude {
	return Claude{credentialsDir: defaultCredentialsDir(".claude"), useFallbacks: true}
}

// NewClaudeFromDir creates an additional Claude account backed by
// dir/.credentials.json.
func NewClaudeFromDir(dir string) Claude {
	return Claude{credentialsDir: expandHome(dir)}
}

func (c Claude) Name() string {
	return accountName("Claude", c.credentialsDir, ".claude")
}

func (c Claude) Fetch(ctx context.Context) Snapshot {
	if auth := readClaudeOAuth(c.credentialsDir); auth.AccessToken != "" {
		if snapshot, err := fetchClaudeOAuth(ctx, auth); err == nil {
			snapshot.Name = c.Name()
			return snapshot
		}
	}
	if c.useFallbacks {
		if key := firstEnv("ANTHROPIC_ADMIN_KEY", "CLAUDE_ADMIN_KEY"); key != "" {
			snapshot := fetchClaudeAdmin(ctx, key)
			snapshot.Name = c.Name()
			return snapshot
		}
		if cookie := firstEnv("QUOTA_CLAUDE_COOKIE", "CLAUDE_COOKIE"); cookie != "" {
			snapshot := fetchClaudeWeb(ctx, cookie)
			snapshot.Name = c.Name()
			return snapshot
		}
	}
	return NewUnavailable(c.Name(), "local auth", "no readable "+filepath.Join(c.credentialsDir, ".credentials.json"))
}

type claudeOAuthAuth struct {
	Path             string
	AccessToken      string
	SubscriptionType string
	RateLimitTier    string
	ExpiresAt        time.Time
}

func readClaudeOAuth(credentialsDir string) claudeOAuthAuth {
	path := filepath.Join(credentialsDir, ".credentials.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return claudeOAuthAuth{}
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return claudeOAuthAuth{}
	}
	oauth, ok := raw["claudeAiOauth"].(map[string]any)
	if !ok {
		return claudeOAuthAuth{}
	}
	token, _ := stringAt(oauth, "accessToken")
	subscription, _ := stringAt(oauth, "subscriptionType")
	tier, _ := stringAt(oauth, "rateLimitTier")
	var expires time.Time
	if ms, ok := number(oauth["expiresAt"]); ok && ms > 0 {
		expires = time.UnixMilli(int64(ms))
	}
	if !expires.IsZero() && time.Now().After(expires) {
		token = ""
	}
	return claudeOAuthAuth{
		Path:             path,
		AccessToken:      token,
		SubscriptionType: subscription,
		RateLimitTier:    tier,
		ExpiresAt:        expires,
	}
}

func fetchClaudeOAuth(ctx context.Context, auth claudeOAuthAuth) (Snapshot, error) {
	body, _, err := get(ctx, "https://api.anthropic.com/api/oauth/usage", map[string]string{
		"Authorization":  "Bearer " + auth.AccessToken,
		"Accept":         "application/json",
		"Content-Type":   "application/json",
		"anthropic-beta": "oauth-2025-04-20",
		"User-Agent":     "claude-code/2.1.0",
	})
	if err != nil {
		return Snapshot{}, err
	}
	value, ok := decodeAny(body)
	if !ok {
		return Snapshot{}, fmt.Errorf("response was not JSON")
	}
	lanes := claudeOAuthLanes(value)
	notes := []string{}
	if !auth.ExpiresAt.IsZero() {
		notes = append(notes, "token expires "+auth.ExpiresAt.Format("Jan 2 15:04"))
	}
	return Snapshot{
		Name:      "Claude",
		Plan:      auth.SubscriptionType,
		Account:   auth.RateLimitTier,
		Source:    "claude-oauth",
		Status:    "ok",
		Lanes:     lanes,
		Notes:     notes,
		UpdatedAt: time.Now(),
	}, nil
}

func claudeOAuthLanes(value any) []Lane {
	var lanes []Lane
	for _, candidate := range []struct {
		label string
		keys  []string
	}{
		{"5h session", []string{"five_hour"}},
		{"7d total", []string{"seven_day"}},
		{"7d OAuth apps", []string{"seven_day_oauth_apps"}},
		{"7d Opus", []string{"seven_day_opus"}},
		{"7d Sonnet", []string{"seven_day_sonnet"}},
		{"Routines", []string{"seven_day_routines", "seven_day_claude_routines", "claude_routines", "routines", "routine", "seven_day_cowork", "cowork"}},
	} {
		for _, key := range candidate.keys {
			window, ok := windowAt(value, key)
			if !ok {
				continue
			}
			if lane, ok := utilizationLane(candidate.label, window); ok {
				lanes = append(lanes, lane)
				break
			}
		}
	}
	if extra, ok := windowAt(value, "extra_usage"); ok {
		if lane, ok := utilizationLane("Extra", extra); ok {
			lanes = append(lanes, lane)
		}
	}
	lanes = append(lanes, claudeScopedWeeklyLanes(value)...)
	if len(lanes) == 0 {
		lanes = append(lanes, Lane{Label: "Usage", Detail: "fetched; parser needs response shape sample"})
	}
	return lanes
}

// claudeScopedWeeklyLanes handles Anthropic's newer limits-array response.
// Model-specific weekly limits such as Fable are no longer represented by
// seven_day_* fields in that shape.
func claudeScopedWeeklyLanes(value any) []Lane {
	root, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	limits, ok := root["limits"].([]any)
	if !ok {
		return nil
	}

	seen := map[string]bool{}
	var lanes []Lane
	for _, candidate := range limits {
		limit, ok := candidate.(map[string]any)
		if !ok {
			continue
		}
		kind, _ := stringAt(limit, "kind")
		group, _ := stringAt(limit, "group")
		if kind != "weekly_scoped" || group != "weekly" {
			continue
		}
		used, ok := number(limit["percent"])
		if !ok {
			continue
		}
		modelName, _ := stringAt(limit, "scope", "model", "display_name")
		modelName = strings.TrimSpace(modelName)
		if modelName == "" || seen[strings.ToLower(modelName)] {
			continue
		}
		seen[strings.ToLower(modelName)] = true

		reset := ""
		if resetsAt, ok := stringAt(limit, "resets_at"); ok {
			reset = formatClaudeReset(resetsAt)
		}
		detail := fmt.Sprintf("%.1f%% used", used)
		if reset != "" {
			detail += " reset " + reset
		}
		lanes = append(lanes, Lane{
			Label:   "7d " + modelName,
			Used:    used,
			Limit:   100,
			Percent: percent(used, 100),
			Unit:    "%",
			Detail:  detail,
			Reset:   reset,
		})
	}
	return lanes
}

func utilizationLane(label string, window map[string]any) (Lane, bool) {
	utilization, ok := number(window["utilization"])
	if !ok {
		return Lane{}, false
	}
	detail := fmt.Sprintf("%.1f%% used", utilization)
	reset := ""
	if value, ok := stringAt(window, "resets_at"); ok {
		reset = formatClaudeReset(value)
		detail += " reset " + reset
	}
	return Lane{
		Label:   label,
		Used:    utilization,
		Limit:   100,
		Percent: percent(utilization, 100),
		Unit:    "%",
		Detail:  detail,
		Reset:   reset,
	}, true
}

func formatClaudeReset(value string) string {
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.999999-07:00",
	} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.Local().Format("Jan 2 15:04")
		}
	}
	return value
}

func fetchClaudeAdmin(ctx context.Context, key string) Snapshot {
	now := time.Now().UTC()
	start := now.AddDate(0, 0, -30)
	query := url.Values{}
	query.Set("starting_at", start.Format(time.RFC3339))
	query.Set("ending_at", now.Format(time.RFC3339))
	query.Set("bucket_width", "1d")
	query.Set("limit", "31")
	query.Add("group_by[]", "model")

	body, _, err := get(ctx, "https://api.anthropic.com/v1/organizations/usage_report/messages?"+query.Encode(), map[string]string{
		"anthropic-version": "2023-06-01",
		"x-api-key":         key,
		"Accept":            "application/json",
	})
	if err != nil {
		return NewUnavailable("Claude", "admin-api", err.Error())
	}
	value, ok := decodeAny(body)
	if !ok {
		return NewUnavailable("Claude", "admin-api", "response was not JSON")
	}

	input, _ := firstNumber(value, "input_tokens")
	cacheCreate, _ := firstNumber(value, "cache_creation_input_tokens")
	cacheRead, _ := firstNumber(value, "cache_read_input_tokens")
	output, _ := firstNumber(value, "output_tokens")
	total := input + cacheCreate + cacheRead + output

	cost := fetchClaudeCost(ctx, key, start, now)
	lanes := []Lane{
		{Label: "30d tokens", Used: total, Detail: compactNumber(total) + " tokens"},
	}
	if cost > 0 {
		lanes = append(lanes, Lane{Label: "30d cost", Used: cost, Detail: fmt.Sprintf("$%.2f", cost)})
	}

	return Snapshot{
		Name:      "Claude",
		Source:    "admin-api",
		Status:    "ok",
		Lanes:     lanes,
		Notes:     []string{"Admin API reports usage but does not expose subscription message caps."},
		UpdatedAt: time.Now(),
	}
}

func fetchClaudeCost(ctx context.Context, key string, start, end time.Time) float64 {
	query := url.Values{}
	query.Set("starting_at", start.Format(time.RFC3339))
	query.Set("ending_at", end.Format(time.RFC3339))
	query.Set("bucket_width", "1d")
	query.Set("limit", "31")
	query.Add("group_by[]", "description")
	body, _, err := get(ctx, "https://api.anthropic.com/v1/organizations/cost_report?"+query.Encode(), map[string]string{
		"anthropic-version": "2023-06-01",
		"x-api-key":         key,
		"Accept":            "application/json",
	})
	if err != nil {
		return 0
	}
	value, ok := decodeAny(body)
	if !ok {
		return 0
	}
	cost, _ := firstNumber(value, "amount", "cost", "cost_usd")
	return cost
}

func fetchClaudeWeb(ctx context.Context, cookie string) Snapshot {
	body, _, err := get(ctx, "https://claude.ai/api/organizations", map[string]string{
		"Cookie": normalizeClaudeCookie(cookie),
	})
	if err != nil {
		return NewUnavailable("Claude", "web", err.Error())
	}
	value, ok := decodeAny(body)
	if !ok {
		return NewUnavailable("Claude", "web", "organizations response was not JSON")
	}
	orgID, account := pickClaudeOrg(value)
	if orgID == "" {
		return NewUnavailable("Claude", "web", "could not find Claude organization id")
	}

	usage, _, err := get(ctx, "https://claude.ai/api/organizations/"+url.PathEscape(orgID)+"/usage", map[string]string{
		"Cookie": normalizeClaudeCookie(cookie),
	})
	if err != nil {
		return NewUnavailable("Claude", "web", err.Error())
	}
	usageJSON, ok := decodeAny(usage)
	if !ok {
		return NewUnavailable("Claude", "web", "usage response was not JSON")
	}

	lanes := claudeWebLanes(usageJSON)
	return Snapshot{
		Name:      "Claude",
		Account:   account,
		Source:    "web",
		Status:    "ok",
		Lanes:     lanes,
		UpdatedAt: time.Now(),
	}
}

func normalizeClaudeCookie(cookie string) string {
	if cookie == "" || len(cookie) > 11 && cookie[:11] == "sessionKey=" {
		return cookie
	}
	return "sessionKey=" + cookie
}

func pickClaudeOrg(value any) (string, string) {
	var orgs []any
	if arr, ok := value.([]any); ok {
		orgs = arr
	} else {
		return "", ""
	}
	for _, candidate := range orgs {
		if org, ok := candidate.(map[string]any); ok {
			id, _ := stringAt(org, "uuid")
			if id == "" {
				id, _ = stringAt(org, "id")
			}
			name, _ := stringAt(org, "name")
			if id != "" {
				return id, name
			}
		}
	}
	return "", ""
}

func claudeWebLanes(value any) []Lane {
	candidates := []struct {
		label string
		keys  []string
	}{
		{"5h session", []string{"five_hour", "five_hour_percent", "session_percent_used"}},
		{"7d messages", []string{"seven_day", "seven_day_percent", "weekly_percent_used"}},
		{"Opus", []string{"seven_day_opus", "opus_percent_used"}},
	}
	var lanes []Lane
	for _, candidate := range candidates {
		used, ok := firstNumber(value, candidate.keys...)
		if !ok {
			continue
		}
		lanes = append(lanes, Lane{
			Label:   candidate.label,
			Used:    used,
			Limit:   100,
			Percent: percent(used, 100),
			Unit:    "%",
			Detail:  fmt.Sprintf("%.1f%% used", used),
		})
	}
	if len(lanes) == 0 {
		lanes = append(lanes, Lane{Label: "Usage", Detail: "fetched; parser needs response shape sample"})
	}
	return lanes
}

func compactNumber(value float64) string {
	if value >= 1_000_000 {
		return fmt.Sprintf("%.1fM", value/1_000_000)
	}
	if value >= 1_000 {
		return fmt.Sprintf("%.1fk", value/1_000)
	}
	return fmt.Sprintf("%.0f", value)
}
