package rig_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/kybernetes-systems/terraform-provider-gastown/internal/gastown/rig"
)

// fakeRunner records GT calls and returns canned output.
type fakeRunner struct {
	calls  [][]string
	output map[string]string // key = joined args prefix, value = stdout
	errs   map[string]error
}

func (f *fakeRunner) GT(_ context.Context, args ...string) (string, error) {
	f.calls = append(f.calls, args)
	key := strings.Join(args, " ")
	for prefix, out := range f.output {
		if strings.HasPrefix(key, prefix) {
			return out, nil
		}
	}
	for prefix, err := range f.errs {
		if strings.HasPrefix(key, prefix) {
			return "", err
		}
	}
	return "", nil
}

func (f *fakeRunner) BD(_ context.Context, _ ...string) (string, error) { return "", nil }

func (f *fakeRunner) HQPath() string {
	return hqPath
}

func (f *fakeRunner) calledWith(prefix ...string) bool {
	want := strings.Join(prefix, " ")
	for _, call := range f.calls {
		if strings.HasPrefix(strings.Join(call, " "), want) {
			return true
		}
	}
	return false
}

var rigAttrTypes = map[string]tftypes.Type{
	"id":           tftypes.String,
	"hq_path":      tftypes.String,
	"name":         tftypes.String,
	"repo":         tftypes.String,
	"runtime":      tftypes.String,
	"max_polecats": tftypes.Number,
	"status":       tftypes.String,
	"prefix":       tftypes.String,
}

func newRigWithRunner(runner *fakeRunner) resource.Resource {
	r := rig.New()
	r.(resource.ResourceWithConfigure).Configure(
		context.Background(),
		resource.ConfigureRequest{ProviderData: runner},
		&resource.ConfigureResponse{},
	)
	return r
}

func getSchema(r resource.Resource) resource.SchemaResponse {
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	return resp
}

func rigPlan(t *testing.T, r resource.Resource, attrs map[string]tftypes.Value) tfsdk.Plan {
	t.Helper()
	s := getSchema(r)
	raw := tftypes.NewValue(tftypes.Object{AttributeTypes: rigAttrTypes}, attrs)
	return tfsdk.Plan{Raw: raw, Schema: s.Schema}
}

func rigState(t *testing.T, r resource.Resource, attrs map[string]tftypes.Value) tfsdk.State {
	t.Helper()
	s := getSchema(r)
	raw := tftypes.NewValue(tftypes.Object{AttributeTypes: rigAttrTypes}, attrs)
	return tfsdk.State{Raw: raw, Schema: s.Schema}
}

const hqPath = "/tmp/gt-test-hq"
const rigName = "testrig"
const repoURL = "https://github.com/example/repo.git"

func baseAttrs() map[string]tftypes.Value {
	return map[string]tftypes.Value{
		"id":           tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"hq_path":      tftypes.NewValue(tftypes.String, hqPath),
		"name":         tftypes.NewValue(tftypes.String, rigName),
		"repo":         tftypes.NewValue(tftypes.String, repoURL),
		"runtime":      tftypes.NewValue(tftypes.String, "claude"),
		"max_polecats": tftypes.NewValue(tftypes.Number, nil),
		"status":       tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"prefix":       tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	}
}

// Test 1: Create calls gt rig add with name and repo.
func TestRigResource_Create_callsRigAdd(t *testing.T) {
	fake := &fakeRunner{}
	r := newRigWithRunner(fake)
	plan := rigPlan(t, r, baseAttrs())
	s := getSchema(r)
	emptyState := tfsdk.State{Raw: tftypes.NewValue(plan.Raw.Type(), nil), Schema: s.Schema}

	var resp resource.CreateResponse
	resp.State = emptyState
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Create returned errors: %v", resp.Diagnostics)
	}
	if !fake.calledWith("rig", "add", rigName, repoURL) {
		t.Fatalf("expected gt rig add %s %s call, got: %v", rigName, repoURL, fake.calls)
	}
}

// Test 2: Create sets runtime via gt rig config set.
func TestRigResource_Create_setsRuntime(t *testing.T) {
	fake := &fakeRunner{}
	r := newRigWithRunner(fake)
	plan := rigPlan(t, r, baseAttrs())
	s := getSchema(r)
	emptyState := tfsdk.State{Raw: tftypes.NewValue(plan.Raw.Type(), nil), Schema: s.Schema}

	var resp resource.CreateResponse
	resp.State = emptyState
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &resp)

	if !fake.calledWith("rig", "config", "set", rigName, "runtime", "claude") {
		t.Fatalf("expected gt rig config set runtime call, got: %v", fake.calls)
	}
}

// Test 3: Delete calls gt rig stop then gt rig dock (in that order).
func TestRigResource_Delete_stopsAndDocks(t *testing.T) {
	fake := &fakeRunner{}
	r := newRigWithRunner(fake)

	attrs := baseAttrs()
	attrs["id"] = tftypes.NewValue(tftypes.String, hqPath+"/"+rigName)
	attrs["status"] = tftypes.NewValue(tftypes.String, "operational")
	attrs["prefix"] = tftypes.NewValue(tftypes.String, "tr")
	state := rigState(t, r, attrs)

	var resp resource.DeleteResponse
	resp.State = state
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete returned errors: %v", resp.Diagnostics)
	}
	if !fake.calledWith("rig", "stop", rigName) {
		t.Fatalf("expected gt rig stop %s call, got: %v", rigName, fake.calls)
	}
	if !fake.calledWith("rig", "dock", rigName) {
		t.Fatalf("expected gt rig dock %s call, got: %v", rigName, fake.calls)
	}

	// stop must come before dock
	var stopIdx, dockIdx int
	for i, call := range fake.calls {
		joined := strings.Join(call, " ")
		if strings.HasPrefix(joined, "rig stop") {
			stopIdx = i
		}
		if strings.HasPrefix(joined, "rig dock") {
			dockIdx = i
		}
	}
	if stopIdx >= dockIdx {
		t.Fatal("expected gt rig stop to be called before gt rig dock")
	}
}

// Test 4: Update calls gt rig config set for runtime.
func TestRigResource_Update_runtime(t *testing.T) {
	fake := &fakeRunner{}
	r := newRigWithRunner(fake)

	attrs := baseAttrs()
	attrs["id"] = tftypes.NewValue(tftypes.String, hqPath+"/"+rigName)
	attrs["status"] = tftypes.NewValue(tftypes.String, "operational")
	attrs["prefix"] = tftypes.NewValue(tftypes.String, "tr")
	attrs["runtime"] = tftypes.NewValue(tftypes.String, "gemini")
	plan := rigPlan(t, r, attrs)
	state := rigState(t, r, attrs)

	var resp resource.UpdateResponse
	resp.State = state
	r.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Update returned errors: %v", resp.Diagnostics)
	}
	if !fake.calledWith("rig", "config", "set", rigName, "runtime", "gemini") {
		t.Fatalf("expected gt rig config set runtime gemini call, got: %v", fake.calls)
	}
}
