package provider

import (
	"context"
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
