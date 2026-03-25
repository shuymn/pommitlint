package main

import (
	"encoding/json"
	"errors"
	"os/exec"
	"testing"
)

type jsonReportResult struct {
	Source string `json:"source"`
	Valid  bool   `json:"valid"`
}

func TestBinaryArgsReplay(t *testing.T) {
	command := exec.CommandContext(
		t.Context(),
		"go",
		"run",
		".",
		"lint",
		"--message",
		"feat: add parser",
		"--format",
		"json",
	)
	command.Dir = "."

	output, err := command.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			t.Fatalf("go run lint: %v\nstderr: %s", err, exitErr.Stderr)
		}
		t.Fatalf("go run lint: %v", err)
	}

	var report jsonReportResult
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}

	if report.Source != "message" {
		t.Fatalf("Source = %q, want %q", report.Source, "message")
	}
	if !report.Valid {
		t.Fatal("Valid = false, want true")
	}
}
