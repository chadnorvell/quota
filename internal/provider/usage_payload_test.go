package provider

import "testing"

func TestCodexOAuthLanesIncludesSecondaryWindow(t *testing.T) {
	payload := map[string]any{
		"rate_limit": map[string]any{
			"primary_window": map[string]any{
				"used_percent":         float64(22),
				"reset_at":             float64(1766948068),
				"limit_window_seconds": float64(7 * 24 * 60 * 60),
			},
			"secondary_window": map[string]any{
				"used_percent":         float64(43),
				"reset_at":             float64(1767407914),
				"limit_window_seconds": float64(5 * 60 * 60),
			},
		},
	}

	lanes := codexOAuthLanes(payload)
	if len(lanes) != 2 {
		t.Fatalf("got %d lanes, want primary and weekly: %+v", len(lanes), lanes)
	}
	if got := lanes[0]; got.Label != "weekly limit" || got.Percent != 22 {
		t.Fatalf("weekly lane = %+v", got)
	}
	if got := lanes[1]; got.Label != "5h limit" || got.Percent != 43 {
		t.Fatalf("5h lane = %+v", got)
	}
}

func TestClaudeUtilizationIsAlreadyAPercentage(t *testing.T) {
	lane, ok := utilizationLane("5h session", map[string]any{"utilization": float64(0.7)})
	if !ok {
		t.Fatal("utilization lane was not parsed")
	}
	if lane.Percent != 0.7 || lane.Used != 0.7 {
		t.Fatalf("session lane = %+v, want 0.7%% used", lane)
	}
}

func TestClaudeOAuthLanesIncludesScopedWeeklyLimits(t *testing.T) {
	payload := map[string]any{
		"five_hour": map[string]any{"utilization": float64(11)},
		"seven_day": map[string]any{"utilization": float64(9)},
		"limits": []any{
			map[string]any{
				"kind":      "session",
				"group":     "session",
				"percent":   float64(11),
				"scope":     nil,
				"is_active": true,
			},
			map[string]any{
				"kind":      "weekly_scoped",
				"group":     "weekly",
				"percent":   float64(5),
				"resets_at": "2026-07-08T09:00:00.283070+00:00",
				"scope": map[string]any{
					"model": map[string]any{"id": nil, "display_name": "Fable"},
				},
				"is_active": false,
			},
		},
	}

	lanes := claudeOAuthLanes(payload)
	if len(lanes) != 3 {
		t.Fatalf("got %d lanes, want session, weekly, and Fable: %+v", len(lanes), lanes)
	}
	if got := lanes[2]; got.Label != "7d Fable" || got.Percent != 5 || got.Reset == "" {
		t.Fatalf("Fable lane = %+v", got)
	}
}

func TestClaudeScopedWeeklyLanesIgnoresUnscopedAndDuplicateLimits(t *testing.T) {
	payload := map[string]any{"limits": []any{
		map[string]any{"kind": "weekly_all", "group": "weekly", "percent": float64(20)},
		map[string]any{
			"kind": "weekly_scoped", "group": "weekly", "percent": float64(1),
			"scope": map[string]any{"model": map[string]any{"display_name": "Fable"}},
		},
		map[string]any{
			"kind": "weekly_scoped", "group": "weekly", "percent": float64(8),
			"scope": map[string]any{"model": map[string]any{"display_name": "fable"}},
		},
		map[string]any{"kind": "weekly_scoped", "group": "weekly", "percent": float64(9)},
	}}

	lanes := claudeScopedWeeklyLanes(payload)
	if len(lanes) != 1 || lanes[0].Percent != 1 {
		t.Fatalf("scoped lanes = %+v", lanes)
	}
}
