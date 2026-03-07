package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	tfexec "github.com/kybernetes-systems/terraform-provider-gastown/internal/exec"
)

// FakeRunner implements tfexec.Runner for tests.
// It mocks all gt/bd commands without spawning real processes.
// It creates minimal filesystem artifacts so resource Read functions work.
type FakeRunner struct {
	hqPath string
	rigs   map[string]*fakeRig
	crews  map[string]*fakeCrew // key: "rigName/crewName"
}

type fakeRig struct {
	repo    string
	runtime string
}

type fakeCrew struct {
	rig  string
	name string
	role string
}

// NewFakeRunner returns a FakeRunner for hqPath.
func NewFakeRunner(hqPath string) tfexec.Runner {
	return &FakeRunner{
		hqPath: hqPath,
		rigs:   make(map[string]*fakeRig),
		crews:  make(map[string]*fakeCrew),
	}
}

func (r *FakeRunner) HQPath() string { return r.hqPath }

func (r *FakeRunner) GT(ctx context.Context, args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("gt: no command")
	}
	switch args[0] {
	case "install":
		return r.gtInstall(args[1:])
	case "down":
		return "ok", nil
	case "uninstall":
		return "ok", nil
	case "rig":
		return r.gtRig(args[1:])
	case "crew":
		return r.gtCrew(args[1:])
	default:
		return "", fmt.Errorf("gt: unknown command %q", args[0])
	}
}

func (r *FakeRunner) BD(ctx context.Context, args ...string) (string, error) {
	return "ok", nil
}

// gtInstall mocks: gt install <path> [flags...]
// Creates mayor/town.json so HQResource.Read does not remove state.
func (r *FakeRunner) gtInstall(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("gt install: missing path")
	}
	hqPath := args[0]
	mayorDir := filepath.Join(hqPath, "mayor")
	if err := os.MkdirAll(mayorDir, 0755); err != nil {
		return "", fmt.Errorf("gt install: %w", err)
	}
	townJSON := filepath.Join(mayorDir, "town.json")
	data := []byte(`{"name":"test-hq","version":"0.0.0"}`)
	if err := os.WriteFile(townJSON, data, 0644); err != nil {
		return "", fmt.Errorf("gt install: %w", err)
	}
	return "HQ installed", nil
}

// gtRig dispatches rig subcommands
func (r *FakeRunner) gtRig(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("gt rig: missing subcommand")
	}
	switch args[0] {
	case "add":
		return r.rigAdd(args[1:])
	case "config":
		return r.rigConfig(args[1:])
	case "status":
		return r.rigStatus(args[1:])
	case "stop":
		return r.rigStop(args[1:])
	case "dock":
		return r.rigDock(args[1:])
	default:
		return "", fmt.Errorf("gt rig: unknown subcommand %q", args[0])
	}
}

func (r *FakeRunner) rigAdd(args []string) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("gt rig add: need <name> <repo>")
	}
	r.rigs[args[0]] = &fakeRig{repo: args[1], runtime: "claude"}
	return "rig created", nil
}

func (r *FakeRunner) rigConfig(args []string) (string, error) {
	// Expects: set <name> <key> <value>
	if len(args) < 4 || args[0] != "set" {
		return "", fmt.Errorf("gt rig config: usage: set <name> <key> <value>")
	}
	rig, ok := r.rigs[args[1]]
	if !ok {
		return "", &tfexec.NotFoundError{Resource: "rig", Name: args[1]}
	}
	if args[2] == "runtime" {
		rig.runtime = args[3]
	}
	return "ok", nil
}

func (r *FakeRunner) rigStatus(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("gt rig status: missing name")
	}
	rig, ok := r.rigs[args[0]]
	if !ok {
		return "", &tfexec.NotFoundError{Resource: "rig", Name: args[0]}
	}
	out, _ := json.Marshal(map[string]interface{}{
		"name":              args[0],
		"repo":              rig.repo,
		"runtime":           rig.runtime,
		"polecats":          0,
		"Beads prefix":      "",
	})
	return string(out), nil
}

func (r *FakeRunner) rigStop(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("gt rig stop: missing name")
	}
	if _, ok := r.rigs[args[0]]; !ok {
		return "", &tfexec.NotFoundError{Resource: "rig", Name: args[0]}
	}
	return "stopped", nil
}

func (r *FakeRunner) rigDock(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("gt rig dock: missing name")
	}
	if _, ok := r.rigs[args[0]]; !ok {
		return "", &tfexec.NotFoundError{Resource: "rig", Name: args[0]}
	}
	delete(r.rigs, args[0])
	return "docked", nil
}

// gtCrew dispatches crew subcommands
func (r *FakeRunner) gtCrew(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("gt crew: missing subcommand")
	}
	switch args[0] {
	case "add":
		return r.crewAdd(args[1:])
	case "list":
		return r.crewList(args[1:])
	case "remove":
		return r.crewRemove(args[1:])
	default:
		return "", fmt.Errorf("gt crew: unknown subcommand %q", args[0])
	}
}

func (r *FakeRunner) crewAdd(args []string) (string, error) {
	// Expects: --rig <rig> <name> <role>
	if len(args) < 4 || args[0] != "--rig" {
		return "", fmt.Errorf("gt crew add: usage: --rig <rig> <name> <role>")
	}
	rigName, name, role := args[1], args[2], args[3]
	if _, ok := r.rigs[rigName]; !ok {
		return "", fmt.Errorf("rig %q not found", rigName)
	}
	r.crews[rigName+"/"+name] = &fakeCrew{rig: rigName, name: name, role: role}
	return "added", nil
}

func (r *FakeRunner) crewList(args []string) (string, error) {
	// Expects: --rig <rig>
	if len(args) < 2 || args[0] != "--rig" {
		return "", fmt.Errorf("gt crew list: usage: --rig <rig>")
	}
	rigName := args[1]
	var list []map[string]string
	for _, c := range r.crews {
		if c.rig == rigName {
			list = append(list, map[string]string{"name": c.name, "role": c.role})
		}
	}
	out, _ := json.Marshal(list)
	return string(out), nil
}

func (r *FakeRunner) crewRemove(args []string) (string, error) {
	// Expects: --rig <rig> --force <name>
	if len(args) < 4 || args[0] != "--rig" || args[2] != "--force" {
		return "", fmt.Errorf("gt crew remove: usage: --rig <rig> --force <name>")
	}
	key := args[1] + "/" + args[3]
	if _, ok := r.crews[key]; !ok {
		return "", &tfexec.NotFoundError{Resource: "crew", Name: args[3]}
	}
	delete(r.crews, key)
	return "removed", nil
}

// Ensure FakeRunner satisfies the Runner interface at compile time.
var _ tfexec.Runner = (*FakeRunner)(nil)
