package service

import (
	"context"
	"sort"
)

// HealthService answers the liveness and readiness use cases for the
// API service by probing its named dependencies (ports).
type HealthService struct {
	dependencies map[string]Pinger
}

// NewHealthService builds a HealthService from named dependency ports.
// Named dependencies keep readiness extensible — storage or the AI
// provider can be added later without a new interface per dependency.
func NewHealthService(dependencies map[string]Pinger) *HealthService {
	return &HealthService{dependencies: dependencies}
}

// CheckResult reports the outcome of one readiness probe.
type CheckResult struct {
	Name   string
	Status string // "ok" or "fail"
	Detail string // failure reason; empty when ok
}

// HealthReport is the readiness use case's output. Checks are sorted
// by name so identical inputs always produce identical outputs.
type HealthReport struct {
	Ready  bool
	Checks []CheckResult
}

// Ready probes every dependency with the caller's deadline and
// aggregates the results. It never reports ready on a nil dependency
// or a probe error — unknown is not ok.
func (s *HealthService) Ready(ctx context.Context) HealthReport {
	report := HealthReport{Ready: true, Checks: make([]CheckResult, 0, len(s.dependencies))}
	for name, dep := range s.dependencies {
		result := CheckResult{Name: name, Status: "ok"}
		if dep == nil {
			result.Status, result.Detail = "fail", "dependency not configured"
		} else if err := dep.Ping(ctx); err != nil {
			result.Status, result.Detail = "fail", err.Error()
		}
		if result.Status == "fail" {
			report.Ready = false
		}
		report.Checks = append(report.Checks, result)
	}
	sort.Slice(report.Checks, func(i, j int) bool {
		return report.Checks[i].Name < report.Checks[j].Name
	})
	return report
}
