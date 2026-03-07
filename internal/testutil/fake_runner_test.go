package testutil

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFakeRunner_HQPath(t *testing.T) {
	r := NewFakeRunner("/tmp/hq")
	if r.HQPath() != "/tmp/hq" {
		t.Errorf("got %q, want /tmp/hq", r.HQPath())
	}
}

func TestFakeRunner_Install_CreatesFilesystem(t *testing.T) {
	hqPath := filepath.Join(t.TempDir(), "hq")
	r := NewFakeRunner(hqPath)

	_, err := r.GT(context.Background(), "install", hqPath, "--git")
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	townJSON := filepath.Join(hqPath, "mayor", "town.json")
	if _, err := os.Stat(townJSON); err != nil {
		t.Errorf("expected mayor/town.json to exist after install: %v", err)
	}
}

func TestFakeRunner_RigLifecycle(t *testing.T) {
	r := NewFakeRunner("/tmp/hq")

	// Add rig
	if _, err := r.GT(context.Background(), "rig", "add", "myrig", "https://github.com/test/repo.git"); err != nil {
		t.Fatalf("rig add: %v", err)
	}

	// Config set
	if _, err := r.GT(context.Background(), "rig", "config", "set", "myrig", "runtime", "gemini"); err != nil {
		t.Fatalf("rig config set: %v", err)
	}

	// Status returns JSON with "polecats"
	out, err := r.GT(context.Background(), "rig", "status", "myrig")
	if err != nil {
		t.Fatalf("rig status: %v", err)
	}
	if !strings.Contains(out, "polecats") {
		t.Error("rig status output missing 'polecats' key")
	}

	// Dock removes the rig
	if _, err := r.GT(context.Background(), "rig", "dock", "myrig"); err != nil {
		t.Fatalf("rig dock: %v", err)
	}

	// Status after dock returns NotFoundError
	_, err = r.GT(context.Background(), "rig", "status", "myrig")
	if err == nil {
		t.Error("expected NotFoundError after dock")
	}
}

func TestFakeRunner_CrewLifecycle(t *testing.T) {
	r := NewFakeRunner("/tmp/hq")

	// Create rig first
	if _, err := r.GT(context.Background(), "rig", "add", "myrig", "https://example.com/repo.git"); err != nil {
		t.Fatalf("rig add: %v", err)
	}

	// Add crew
	if _, err := r.GT(context.Background(), "crew", "add", "--rig", "myrig", "alice", "operator"); err != nil {
		t.Fatalf("crew add: %v", err)
	}

	// List includes alice
	out, err := r.GT(context.Background(), "crew", "list", "--rig", "myrig")
	if err != nil {
		t.Fatalf("crew list: %v", err)
	}
	if !strings.Contains(out, "alice") {
		t.Errorf("crew list missing alice, got: %s", out)
	}

	// Remove alice
	if _, err := r.GT(context.Background(), "crew", "remove", "--rig", "myrig", "--force", "alice"); err != nil {
		t.Fatalf("crew remove: %v", err)
	}

	// Remove non-existent → NotFoundError
	_, err = r.GT(context.Background(), "crew", "remove", "--rig", "myrig", "--force", "alice")
	if err == nil {
		t.Error("expected NotFoundError for missing crew")
	}
}

func TestFakeRunner_BD_Status(t *testing.T) {
	r := NewFakeRunner("/tmp/hq")
	out, err := r.BD(context.Background(), "status")
	if err != nil {
		t.Fatalf("bd status: %v", err)
	}
	if out == "" {
		t.Error("expected non-empty output")
	}
}
