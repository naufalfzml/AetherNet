package usecase

import (
	"context"
	"errors"
	"fmt"
)

type HealthService struct {
	Storage HealthCheck
	DA      HealthCheck
	Compute HealthCheck
	Chain   HealthCheck
}

type DependencyHealth struct {
	Name  string `json:"name"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type HealthReport struct {
	Status       string             `json:"status"`
	Dependencies []DependencyHealth `json:"dependencies"`
}

func (s HealthService) Check(ctx context.Context) HealthReport {
	checks := []struct {
		name   string
		client HealthCheck
	}{
		{name: "storage", client: s.Storage},
		{name: "compute", client: s.Compute},
		{name: "chain", client: s.Chain},
	}

	report := HealthReport{Status: "ok"}
	for _, check := range checks {
		dep := DependencyHealth{Name: check.name, OK: true}
		if check.client == nil {
			dep.OK = false
			dep.Error = "not configured"
		} else if err := check.client.Health(ctx); err != nil {
			dep.OK = false
			dep.Error = err.Error()
		}
		if !dep.OK {
			report.Status = "degraded"
		}
		report.Dependencies = append(report.Dependencies, dep)
	}
	return report
}

func (r HealthReport) Err() error {
	if r.Status == "ok" {
		return nil
	}
	return errors.New("one or more dependencies are unhealthy")
}

type StaticHealthClient struct {
	Name string
	Err  error
}

func (c StaticHealthClient) Health(context.Context) error {
	if c.Err != nil {
		return fmt.Errorf("%s: %w", c.Name, c.Err)
	}
	return nil
}
