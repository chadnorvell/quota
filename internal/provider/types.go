package provider

import (
	"context"
	"strings"
	"time"
)

type Source interface {
	Name() string
	Fetch(context.Context) Snapshot
}

type Snapshot struct {
	Name      string
	Plan      string
	Account   string
	Source    string
	Status    string
	Lanes     []Lane
	Notes     []string
	Error     string
	UpdatedAt time.Time
}

type Lane struct {
	Label    string
	Used     float64
	Limit    float64
	Percent  float64
	Unit     string
	Detail   string
	Reset    string
	Critical bool
}

func NewUnavailable(name, source, err string) Snapshot {
	return Snapshot{
		Name:      name,
		Source:    source,
		Status:    "unavailable",
		Error:     err,
		UpdatedAt: time.Now(),
	}
}

func percent(used, limit float64) float64 {
	if limit <= 0 {
		return 0
	}
	value := used / limit * 100
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func PlanName(raw string) string {
	normalized := strings.TrimSpace(strings.ToLower(raw))
	switch normalized {
	case "prolite":
		return "Pro Lite"
	case "pro":
		return "Pro"
	case "plus":
		return "Plus"
	case "team":
		return "Team"
	case "enterprise":
		return "Enterprise"
	case "max":
		return "Max"
	case "free":
		return "Free"
	}
	if normalized == "" {
		return raw
	}
	parts := strings.FieldsFunc(normalized, func(char rune) bool {
		return char == '_' || char == '-' || char == ' '
	})
	for index, part := range parts {
		if part == "" {
			continue
		}
		parts[index] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}
