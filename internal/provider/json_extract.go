package provider

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

func decodeAny(data []byte) (any, bool) {
	var value any
	if err := json.Unmarshal(data, &value); err == nil {
		return value, true
	}
	return nil, false
}

func numericAt(value any, path ...string) (float64, bool) {
	current := value
	for _, key := range path {
		object, ok := current.(map[string]any)
		if !ok {
			return 0, false
		}
		current, ok = object[key]
		if !ok {
			return 0, false
		}
	}
	return number(current)
}

func stringAt(value any, path ...string) (string, bool) {
	current := value
	for _, key := range path {
		object, ok := current.(map[string]any)
		if !ok {
			return "", false
		}
		current, ok = object[key]
		if !ok {
			return "", false
		}
	}
	if text, ok := current.(string); ok && strings.TrimSpace(text) != "" {
		return text, true
	}
	return "", false
}

func number(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		if !math.IsNaN(typed) && !math.IsInf(typed, 0) {
			return typed, true
		}
	case int:
		return float64(typed), true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	}
	return 0, false
}

func firstNumber(value any, names ...string) (float64, bool) {
	var found float64
	var ok bool
	walk(value, func(key string, candidate any) {
		if ok {
			return
		}
		normalizedKey := normalizeKey(key)
		for _, name := range names {
			if normalizedKey == normalizeKey(name) {
				found, ok = number(candidate)
				return
			}
		}
	})
	return found, ok
}

func walk(value any, visit func(string, any)) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			visit(key, child)
			walk(child, visit)
		}
	case []any:
		for _, child := range typed {
			walk(child, visit)
		}
	}
}

func normalizeKey(value string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + 32
		}
		return -1
	}, value)
}

func htmlPercentLanes(data []byte, labels ...string) []Lane {
	text := string(data)
	var lanes []Lane
	for _, label := range labels {
		pattern := regexp.MustCompile(`(?is)` + regexp.QuoteMeta(label) + `.{0,120}?([0-9]+(?:\.[0-9]+)?)\s*%`)
		match := pattern.FindStringSubmatch(text)
		if len(match) != 2 {
			continue
		}
		value, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			continue
		}
		lanes = append(lanes, Lane{
			Label:   label,
			Used:    value,
			Limit:   100,
			Percent: percent(value, 100),
			Unit:    "%",
			Detail:  fmt.Sprintf("%.1f%% used", value),
		})
	}
	return lanes
}
