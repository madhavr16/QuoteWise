package service

import (
	"context"
	"errors"
	"testing"
)

type fakePinger struct {
	err error
}

func (f fakePinger) Ping(context.Context) error { return f.err }

func TestReadyPassesWhenAllDependenciesPass(t *testing.T) {
	svc := NewHealthService(map[string]Pinger{
		"postgres":   fakePinger{},
		"ai_service": fakePinger{},
	})

	report := svc.Ready(context.Background())
	if !report.Ready {
		t.Error("Ready = false, want true")
	}
	if len(report.Checks) != 2 {
		t.Fatalf("len(Checks) = %d, want 2", len(report.Checks))
	}
	for _, check := range report.Checks {
		if check.Status != "ok" {
			t.Errorf("check %q status = %q, want %q", check.Name, check.Status, "ok")
		}
		if check.Detail != "" {
			t.Errorf("check %q detail = %q, want empty", check.Name, check.Detail)
		}
	}
}

func TestReadyFailsWhenADependencyFails(t *testing.T) {
	svc := NewHealthService(map[string]Pinger{
		"postgres":   fakePinger{err: errors.New("connection refused")},
		"ai_service": fakePinger{},
	})

	report := svc.Ready(context.Background())
	if report.Ready {
		t.Error("Ready = true, want false")
	}
	var postgres CheckResult
	for _, check := range report.Checks {
		if check.Name == "postgres" {
			postgres = check
		}
	}
	if postgres.Status != "fail" {
		t.Errorf("postgres status = %q, want %q", postgres.Status, "fail")
	}
	if postgres.Detail != "connection refused" {
		t.Errorf("postgres detail = %q, want %q", postgres.Detail, "connection refused")
	}
}

func TestReadyFailsOnNilDependency(t *testing.T) {
	svc := NewHealthService(map[string]Pinger{"postgres": nil})

	report := svc.Ready(context.Background())
	if report.Ready {
		t.Error("Ready = true, want false when a dependency is not configured")
	}
	if report.Checks[0].Detail != "dependency not configured" {
		t.Errorf("detail = %q, want %q", report.Checks[0].Detail, "dependency not configured")
	}
}

func TestReadyOutputIsDeterministic(t *testing.T) {
	svc := NewHealthService(map[string]Pinger{
		"postgres":   fakePinger{},
		"ai_service": fakePinger{},
	})

	first := svc.Ready(context.Background())
	// Map iteration order varies between calls; the report must not.
	for i := 0; i < 20; i++ {
		next := svc.Ready(context.Background())
		if next.Ready != first.Ready || len(next.Checks) != len(first.Checks) {
			t.Fatalf("report shape changed on run %d", i)
		}
		for j := range next.Checks {
			if next.Checks[j].Name != first.Checks[j].Name {
				t.Fatalf("check order changed on run %d: %v vs %v", i, next.Checks[j].Name, first.Checks[j].Name)
			}
		}
	}
}
