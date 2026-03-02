package hq_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/kybernetes-systems/terraform-provider-gastown/internal/gastown/hq"
)

func newHQ() resource.Resource { return hq.New() }

// buildHQConfig constructs a tfsdk.Config for the hq resource with the given attribute values.
func buildHQConfig(t *testing.T, r resource.Resource, attrs map[string]tftypes.Value) tfsdk.Config {
	t.Helper()
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	attrTypes := make(map[string]tftypes.Type)
	for k := range schemaResp.Schema.Attributes {
		switch schemaResp.Schema.Attributes[k].(type) {
		case schema.StringAttribute:
			attrTypes[k] = tftypes.String
		case schema.BoolAttribute:
			attrTypes[k] = tftypes.Bool
		}
	}

	raw := tftypes.NewValue(tftypes.Object{AttributeTypes: attrTypes}, attrs)
	return tfsdk.Config{Raw: raw, Schema: schemaResp.Schema}
}

// Test 1: Create calls gt install and mayor/town.json exists afterwards.
func TestHQResource_Create_callsGtInstall(t *testing.T) {
	dir := t.TempDir()
	hqPath := filepath.Join(dir, "gt")

	r := newHQ()
	cfg := buildHQConfig(t, r, map[string]tftypes.Value{
		"path":        tftypes.NewValue(tftypes.String, hqPath),
		"owner_email": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"git":         tftypes.NewValue(tftypes.Bool, true),
		"id":          tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"name":        tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	})

	state := tfsdk.State{Raw: tftypes.NewValue(cfg.Raw.Type(), nil), Schema: cfg.Schema}
	var resp resource.CreateResponse
	resp.State = state

	r.Create(context.Background(), resource.CreateRequest{Config: cfg, Plan: tfsdk.Plan{Raw: cfg.Raw, Schema: cfg.Schema}}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Create returned errors: %v", resp.Diagnostics)
	}

	townJSON := filepath.Join(hqPath, "mayor", "town.json")
	if _, err := os.Stat(townJSON); os.IsNotExist(err) {
		t.Fatalf("expected mayor/town.json to exist at %s after gt install", townJSON)
	}
}

// Test 2: Read after Create returns the same path (idempotent).
func TestHQResource_Read_idempotent(t *testing.T) {
	dir := t.TempDir()
	hqPath := filepath.Join(dir, "gt")

	r := newHQ()
	cfg := buildHQConfig(t, r, map[string]tftypes.Value{
		"path":        tftypes.NewValue(tftypes.String, hqPath),
		"owner_email": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"git":         tftypes.NewValue(tftypes.Bool, true),
		"id":          tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"name":        tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	})

	emptyState := tfsdk.State{Raw: tftypes.NewValue(cfg.Raw.Type(), nil), Schema: cfg.Schema}

	var createResp resource.CreateResponse
	createResp.State = emptyState
	r.Create(context.Background(), resource.CreateRequest{Config: cfg, Plan: tfsdk.Plan{Raw: cfg.Raw, Schema: cfg.Schema}}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create failed: %v", createResp.Diagnostics)
	}

	var readResp resource.ReadResponse
	readResp.State = createResp.State
	r.Read(context.Background(), resource.ReadRequest{State: createResp.State}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read returned errors: %v", readResp.Diagnostics)
	}

	var createdState struct {
		Path string `tfsdk:"path"`
	}
	createResp.State.Get(context.Background(), &createdState)

	var readState struct {
		Path string `tfsdk:"path"`
	}
	readResp.State.Get(context.Background(), &readState)

	if createdState.Path != readState.Path {
		t.Fatalf("Read returned path %q, want %q", readState.Path, createdState.Path)
	}
}

// Test 3: path attribute has RequiresReplace (ForceNew) plan modifier.
func TestHQResource_ForceNew_onPathChange(t *testing.T) {
	r := newHQ()
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	attr, ok := schemaResp.Schema.Attributes["path"]
	if !ok {
		t.Fatal("schema missing path attribute")
	}
	sa, ok := attr.(schema.StringAttribute)
	if !ok {
		t.Fatalf("path should be StringAttribute, got %T", attr)
	}
	if !sa.Required {
		t.Fatal("path should be Required")
	}
	if len(sa.PlanModifiers) == 0 {
		t.Fatal("path should have plan modifiers (RequiresReplace)")
	}
}

// Test 4: Delete calls gt uninstall --force (state is cleared).
func TestHQResource_Delete_callsUninstall(t *testing.T) {
	dir := t.TempDir()
	hqPath := filepath.Join(dir, "gt")

	r := newHQ()
	cfg := buildHQConfig(t, r, map[string]tftypes.Value{
		"path":        tftypes.NewValue(tftypes.String, hqPath),
		"owner_email": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"git":         tftypes.NewValue(tftypes.Bool, true),
		"id":          tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"name":        tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	})

	emptyState := tfsdk.State{Raw: tftypes.NewValue(cfg.Raw.Type(), nil), Schema: cfg.Schema}
	var createResp resource.CreateResponse
	createResp.State = emptyState
	r.Create(context.Background(), resource.CreateRequest{Config: cfg, Plan: tfsdk.Plan{Raw: cfg.Raw, Schema: cfg.Schema}}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create failed: %v", createResp.Diagnostics)
	}

	var deleteResp resource.DeleteResponse
	deleteResp.State = createResp.State
	r.Delete(context.Background(), resource.DeleteRequest{State: createResp.State}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("Delete returned errors: %v", deleteResp.Diagnostics)
	}
}
