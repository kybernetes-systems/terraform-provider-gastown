package crew_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/kybernetes-systems/terraform-provider-gastown/internal/gastown/crew"
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

var crewAttrTypes = map[string]tftypes.Type{
	"id":      tftypes.String,
	"hq_path": tftypes.String,
	"rig":     tftypes.String,
	"name":    tftypes.String,
	"role":    tftypes.String,
}

func newCrewWithRunner(runner *fakeRunner) resource.Resource {
	r := crew.New()
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

func crewPlan(t *testing.T, r resource.Resource, attrs map[string]tftypes.Value) tfsdk.Plan {
	t.Helper()
	s := getSchema(r)
	raw := tftypes.NewValue(tftypes.Object{AttributeTypes: crewAttrTypes}, attrs)
	return tfsdk.Plan{Raw: raw, Schema: s.Schema}
}

func crewState(t *testing.T, r resource.Resource, attrs map[string]tftypes.Value) tfsdk.State {
	t.Helper()
	s := getSchema(r)
	raw := tftypes.NewValue(tftypes.Object{AttributeTypes: crewAttrTypes}, attrs)
	return tfsdk.State{Raw: raw, Schema: s.Schema}
}

const hqPath = "/tmp/gt-test-hq"
const rigName = "testrig"
const crewName = "testcrew"
const crewRole = "operator"

func baseAttrs() map[string]tftypes.Value {
	return map[string]tftypes.Value{
		"id":      tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"hq_path": tftypes.NewValue(tftypes.String, hqPath),
		"rig":     tftypes.NewValue(tftypes.String, rigName),
		"name":    tftypes.NewValue(tftypes.String, crewName),
		"role":    tftypes.NewValue(tftypes.String, crewRole),
	}
}

// Test 1: Create calls gt crew add with rig, name, and role.
func TestCrewResource_Create_callsCrewAdd(t *testing.T) {
	fake := &fakeRunner{}
	r := newCrewWithRunner(fake)
	plan := crewPlan(t, r, baseAttrs())
	s := getSchema(r)
	emptyState := tfsdk.State{Raw: tftypes.NewValue(plan.Raw.Type(), nil), Schema: s.Schema}

	var resp resource.CreateResponse
	resp.State = emptyState
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Create returned errors: %v", resp.Diagnostics)
	}
	if !fake.calledWith("crew", "add", "--rig", rigName, crewName, crewRole) {
		t.Fatalf("expected gt crew add --rig %s %s %s call, got: %v", rigName, crewName, crewRole, fake.calls)
	}
}

// Test 2: Read calls gt crew list and succeeds if crew found.
func TestCrewResource_Read_findsCrew(t *testing.T) {
	fake := &fakeRunner{}
	fake.output = map[string]string{
		"crew list " + rigName: "othercrew\n" + crewName + "\nanothercrew\n",
	}
	r := newCrewWithRunner(fake)
	attrs := baseAttrs()
	attrs["id"] = tftypes.NewValue(tftypes.String, hqPath+"/"+rigName+"/"+crewName)
	state := crewState(t, r, attrs)

	var resp resource.ReadResponse
	resp.State = state
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read returned errors: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsKnown() {
		t.Fatal("expected state to be preserved")
	}
}

// Test 3: Read removes resource if crew NOT found.
func TestCrewResource_Read_removesIfNotFound(t *testing.T) {
	fake := &fakeRunner{}
	fake.output = map[string]string{
		"crew list " + rigName: "othercrew\nanothercrew\n",
	}
	r := newCrewWithRunner(fake)
	attrs := baseAttrs()
	attrs["id"] = tftypes.NewValue(tftypes.String, hqPath+"/"+rigName+"/"+crewName)
	state := crewState(t, r, attrs)

	var resp resource.ReadResponse
	resp.State = state
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read returned errors: %v", resp.Diagnostics)
	}
	if resp.State.Raw.IsKnown() && !resp.State.Raw.IsNull() {
		t.Fatal("expected state to be removed")
	}
}

// Test 4: Delete calls gt crew remove.
func TestCrewResource_Delete_callsCrewRemove(t *testing.T) {
	fake := &fakeRunner{}
	r := newCrewWithRunner(fake)
	attrs := baseAttrs()
	attrs["id"] = tftypes.NewValue(tftypes.String, hqPath+"/"+rigName+"/"+crewName)
	state := crewState(t, r, attrs)

	var resp resource.DeleteResponse
	resp.State = state
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete returned errors: %v", resp.Diagnostics)
	}
	if !fake.calledWith("crew", "remove", "--rig", rigName, "--force", crewName) {
		t.Fatalf("expected gt crew remove --rig %s --force %s call, got: %v", rigName, crewName, fake.calls)
	}
}

// Test 5: Update is not supported and returns an error.
func TestCrewResource_Update_fails(t *testing.T) {
	fake := &fakeRunner{}
	r := newCrewWithRunner(fake)
	attrs := baseAttrs()
	attrs["id"] = tftypes.NewValue(tftypes.String, hqPath+"/"+rigName+"/"+crewName)
	state := crewState(t, r, attrs)
	plan := crewPlan(t, r, attrs)

	var resp resource.UpdateResponse
	resp.State = state
	r.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected Update to return an error diagnostic")
	}
}
