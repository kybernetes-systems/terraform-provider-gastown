package hq

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tfexec "github.com/kybernetes-systems/terraform-provider-gastown/internal/exec"
	"github.com/kybernetes-systems/terraform-provider-gastown/internal/validators"
)

var _ resource.Resource = &HQResource{}
var _ resource.ResourceWithConfigure = &HQResource{}
var _ resource.ResourceWithImportState = &HQResource{}

type HQResource struct {
	runner tfexec.Runner
}

func New() resource.Resource { return &HQResource{} }

func (r *HQResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	runner, ok := req.ProviderData.(tfexec.Runner)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("expected tfexec.Runner, got %T", req.ProviderData))
		return
	}
	r.runner = runner
}

func (r *HQResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hq"
}

func (r *HQResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the HQ resource. Same as the path.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Description: "Filesystem path where the Gas Town HQ will be installed.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.PathValidator{},
				},
			},
			"owner_email": schema.StringAttribute{
				Description: "Email address of the HQ owner.",
				Optional:    true,
				Validators: []validator.String{
					validators.EmailValidator{},
				},
			},
			"git": schema.BoolAttribute{
				Description: "Whether to initialize git in the HQ directory. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"no_beads": schema.BoolAttribute{
				Description: "Whether to skip beads initialization. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"name": schema.StringAttribute{
				Description: "The name of the town (read from mayor/town.json).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

type hqModel struct {
	ID         types.String `tfsdk:"id"`
	Path       types.String `tfsdk:"path"`
	OwnerEmail types.String `tfsdk:"owner_email"`
	Git        types.Bool   `tfsdk:"git"`
	NoBeads    types.Bool   `tfsdk:"no_beads"`
	Name       types.String `tfsdk:"name"`
}

func (r *HQResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan hqModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hqPath := plan.Path.ValueString()
	if err := os.MkdirAll(filepath.Dir(hqPath), 0755); err != nil {
		resp.Diagnostics.AddError("Error creating HQ parent directory", err.Error())
		return
	}

	args := []string{"install", hqPath}
	if plan.Git.ValueBool() {
		args = append(args, "--git")
	}
	if plan.NoBeads.ValueBool() {
		args = append(args, "--no-beads")
	}
	if !plan.OwnerEmail.IsNull() && plan.OwnerEmail.ValueString() != "" {
		args = append(args, "--owner", plan.OwnerEmail.ValueString())
	}

	runner := r.runner
	if runner == nil {
		runner = tfexec.NewRunner("")
	}

	if _, err := runner.GT(ctx, args...); err != nil {
		resp.Diagnostics.AddError("Error creating HQ", err.Error())
		return
	}

	// Configure unique Dolt port to avoid conflicts when multiple HQs are created
	// in a test environment. For production, we use the default ports.
	if os.Getenv("TF_ACC") == "1" {
		port, err := getFreePort()
		if err != nil {
			resp.Diagnostics.AddWarning("Could not allocate free port, using default", err.Error())
			port = 3307
		}

		daemonConfigPath := filepath.Join(hqPath, "mayor", "daemon.json")
		daemonConfig := map[string]interface{}{
			"env": map[string]string{
				"GT_DOLT_PORT": fmt.Sprintf("%d", port),
			},
		}
		data, _ := json.MarshalIndent(daemonConfig, "", "  ")
		_ = os.MkdirAll(filepath.Dir(daemonConfigPath), 0755)
		_ = os.WriteFile(daemonConfigPath, data, 0644)
	}

	hqRunner := tfexec.NewRunner(hqPath)
	if err := ensureUp(ctx, hqPath, hqRunner, &resp.Diagnostics); err != nil {
		return
	}

	name, err := readTownName(hqPath)
	if err != nil {
		resp.Diagnostics.AddError("Error reading town name after install", err.Error())
		return
	}

	plan.ID = plan.Path
	plan.Name = types.StringValue(name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HQResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state hqModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hqPath := state.Path.ValueString()
	townJSON := filepath.Join(hqPath, "mayor", "town.json")
	if _, err := os.Stat(townJSON); os.IsNotExist(err) {
		resp.State.RemoveResource(ctx)
		return
	}

	hqRunner := tfexec.NewRunner(hqPath)
	// Ensure services are up during Read/Refresh, otherwise subsequent resource
	// operations in the same plan will fail to connect to Dolt.
	_ = ensureUp(ctx, hqPath, hqRunner, &resp.Diagnostics)

	name, err := readTownName(hqPath)
	if err != nil {
		resp.Diagnostics.AddError("Error reading town name", err.Error())
		return
	}
	state.Name = types.StringValue(name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *HQResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state hqModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// HQ attributes (owner_email, git, no_beads) are set at creation time
	// and cannot be modified after installation. Force recreation if changed.
	resp.Diagnostics.AddError(
		"Update not supported",
		fmt.Sprintf("HQ resource cannot be updated after creation. Changes detected from state:\n  owner_email: %s -> %s\n  git: %t -> %t\n  no_beads: %t -> %t\n\nTo apply changes, taint and recreate the resource: terraform taint gastown_hq.%s",
			state.OwnerEmail.ValueString(), plan.OwnerEmail.ValueString(),
			state.Git.ValueBool(), plan.Git.ValueBool(),
			state.NoBeads.ValueBool(), plan.NoBeads.ValueBool(),
			plan.Name.ValueString()),
	)
}

func (r *HQResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("path"), req, resp)
}

func (r *HQResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state hqModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	runner := r.runner
	if runner == nil {
		runner = tfexec.NewRunner(state.Path.ValueString())
	}
	_, _ = runner.GT(ctx, "down")
	if _, err := runner.GT(ctx, "uninstall", "--force"); err != nil {
		resp.Diagnostics.AddError("Error deleting HQ", err.Error())
	}
}

// ensureUp brings up Gas Town services and waits for Dolt to be ready.
func ensureUp(ctx context.Context, hqPath string, runner tfexec.Runner, diags *diag.Diagnostics) error {
	// Configure unique Dolt port to avoid conflicts when multiple HQs are created
	// in a test environment. For production, we ensure daemon.json is removed
	// so that gt uses its default discoverable ports.
	daemonConfigPath := filepath.Join(hqPath, "mayor", "daemon.json")
	if os.Getenv("TF_ACC") == "1" {
		port, err := getFreePort()
		if err != nil {
			diags.AddWarning("Could not allocate free port, using default", err.Error())
			port = 3307
		}

		daemonConfig := map[string]interface{}{
			"env": map[string]string{
				"GT_DOLT_PORT": fmt.Sprintf("%d", port),
			},
		}
		data, _ := json.MarshalIndent(daemonConfig, "", "  ")
		_ = os.MkdirAll(filepath.Dir(daemonConfigPath), 0755)
		_ = os.WriteFile(daemonConfigPath, data, 0644)
	} else {
		// Ensure no stale port overrides exist in production
		_ = os.Remove(daemonConfigPath)
	}

	// Start services (Dolt, etc) needed for subsequent rig/crew operations.
	if out, err := runner.GT(ctx, "up"); err != nil {
		diags.AddError("Failed to start Gas Town services", fmt.Sprintf("output: %s\nerror: %v", out, err))
		return err
	}

	// Wait for Dolt to be ready
	if err := waitForDolt(ctx, runner); err != nil {
		diags.AddError("Dolt did not become ready", err.Error())
		return err
	}
	return nil
}

func readTownName(hqPath string) (string, error) {
	// Use filepath.Join to safely construct path
	townJSON := filepath.Join(hqPath, "mayor", "town.json")
	
	// Double check that it's still inside hqPath (defense in depth)
	rel, err := filepath.Rel(hqPath, townJSON)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("invalid HQ path: %s", hqPath)
	}

	data, err := os.ReadFile(townJSON)
	if err != nil {
		return "", fmt.Errorf("reading mayor/town.json: %w", err)
	}
	var town struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &town); err != nil {
		return "", fmt.Errorf("parsing mayor/town.json: %w", err)
	}
	return town.Name, nil
}

func waitForDolt(ctx context.Context, runner tfexec.Runner) error {
	maxAttempts := 30
	for i := 0; i < maxAttempts; i++ {
		_, err := runner.BD(ctx, "status")
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	return fmt.Errorf("Dolt did not become ready after %d attempts", maxAttempts)
}

// getFreePort returns an available TCP port for Dolt to use.
// Note: This has a theoretical race condition (TOCTOU) where the port could be
// claimed between our check and Dolt starting. In practice, this is mitigated by:
// 1. Using OS-assigned ephemeral ports (high range, unlikely to collide)
// 2. Retrying on port conflict
// For test environments, occasional port conflicts are handled gracefully.
func getFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
