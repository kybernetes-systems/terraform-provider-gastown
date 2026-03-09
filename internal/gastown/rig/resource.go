package rig

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tfexec "github.com/kybernetes-systems/terraform-provider-gastown/internal/exec"
	"github.com/kybernetes-systems/terraform-provider-gastown/internal/validators"
)

var _ resource.Resource = &RigResource{}
var _ resource.ResourceWithConfigure = &RigResource{}
var _ resource.ResourceWithImportState = &RigResource{}

type RigResource struct {
	runner tfexec.Runner
}

func New() resource.Resource { return &RigResource{} }

func (r *RigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rig"
}

func (r *RigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Gas Town rig—a declarative agent workspace that executes tasks through " +
			"a Git-backed operational model. Rigs combine a source repository, runtime configuration, and " +
			"worker pool (polecats) into a reproducible execution environment. " +
			"**Important**: Destroying a rig resource calls `gt rig stop` followed by `gt rig dock`—the rig " +
			"directory and its operational history are preserved on disk. To fully remove a rig, use " +
			"`gt rig remove` outside of Terraform.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the rig resource, constructed as the join of hq_path and rig name. " +
					"This computed value remains stable throughout the resource lifecycle and is used for internal resource tracking.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"hq_path": schema.StringAttribute{
				Description: "Absolute path to the Gas Town HQ directory where this rig will be registered and operated. " +
					"The rig's operational state, configuration, and worktrees are stored within this HQ. " +
					"Changing this value after creation forces replacement of the rig—Terraform will destroy the existing " +
					"rig (stopping and docking it) and create a new one in the specified HQ.",
				Required: true,
				Validators: []validator.String{
					validators.PathValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Unique name for the rig within this HQ, used as the primary identifier for all rig operations. " +
					"Set once at creation; changing this value forces replacement of the resource (the existing rig is stopped " +
					"and docked, a new one is created). Corresponds to the <name> argument of `gt rig add`. " +
					"Must be a valid safe name containing only alphanumeric characters, hyphens, and underscores.",
				Required: true,
				Validators: []validator.String{
					validators.SafeNameValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"repo": schema.StringAttribute{
				Description: "Git repository URL or local filesystem path that serves as the source for this rig. " +
					"The repository is cloned into the rig's workspace and defines the available agent configurations, " +
					"workflows, and operational code. Changing this value after creation forces replacement of the rig. " +
					"Supports HTTPS URLs, SSH URLs (git@host:path format), and local absolute paths.",
				Required: true,
				Validators: []validator.String{
					validators.RepoURLValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"runtime": schema.StringAttribute{
				Description: "Runtime environment that determines how the rig interprets and executes tasks. " +
					"Common values include 'claude' for standard Claude Code operations. The runtime affects " +
					"available capabilities, environment setup, and execution constraints. Can be modified after " +
					"creation without replacement via `gt rig config set`. Defaults to the Gas Town system default of 'claude'.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("claude"),
				Validators: []validator.String{
					validators.SafeNameValidator{},
				},
			},
			"max_polecats": schema.Int64Attribute{
				Description: "Maximum number of polecats (concurrent workers) that this rig may spawn for task execution. " +
					"Higher values enable more parallelism but consume more system resources. For test environments, " +
					"consider setting this to 0 to prevent worker spawning. Can be modified after creation without replacement. " +
					"Defaults to the Gas Town system default of 3. Changing this value after creation forces replacement " +
					"to ensure clean worker pool initialization.",
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(3),
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Description: "Current operational status of the rig as reported by Gas Town, such as 'operational', " +
					"'parked', or 'docked'. This computed value reflects the actual state of the rig process and " +
					"is updated during each Terraform refresh. A status of 'docked' indicates the rig is persistently " +
					"offline and will not auto-restart.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"prefix": schema.StringAttribute{
				Description: "Beads prefix assigned to this rig, read from `gt rig status` output during refresh. " +
					"This prefix determines the naming convention for issues and beads created by this rig's crews. " +
					"Computed at creation time and updated during refresh to reflect any external configuration changes. " +
					"The prefix includes a trailing hyphen (e.g., 'rig-name-') which is stripped in this attribute.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

type rigModel struct {
	ID          types.String `tfsdk:"id"`
	HQPath      types.String `tfsdk:"hq_path"`
	Name        types.String `tfsdk:"name"`
	Repo        types.String `tfsdk:"repo"`
	Runtime     types.String `tfsdk:"runtime"`
	MaxPolecats types.Int64  `tfsdk:"max_polecats"`
	Status      types.String `tfsdk:"status"`
	Prefix      types.String `tfsdk:"prefix"`
}

func (r *RigResource) runner_(hqPath string) tfexec.Runner {
	if r.runner != nil {
		return r.runner
	}
	return tfexec.NewRunner(hqPath)
}

func (r *RigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan rigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	runner := r.runner_(plan.HQPath.ValueString())

	if _, err := runner.GT(ctx, "rig", "add", plan.Name.ValueString(), plan.Repo.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error creating rig", err.Error())
		return
	}

	if !plan.Runtime.IsNull() && !plan.Runtime.IsUnknown() && plan.Runtime.ValueString() != "" {
		if _, err := runner.GT(ctx, "rig", "config", "set", plan.Name.ValueString(), "runtime", plan.Runtime.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error setting rig runtime", err.Error())
			return
		}
	}

	// Set max_polecats to prevent test rigs from spawning workers (ADR 0011)
	if !plan.MaxPolecats.IsNull() && !plan.MaxPolecats.IsUnknown() {
		maxPolecats := fmt.Sprintf("%d", plan.MaxPolecats.ValueInt64())
		if _, err := runner.GT(ctx, "rig", "config", "set", plan.Name.ValueString(), "max_polecats", maxPolecats); err != nil {
			resp.Diagnostics.AddError("Error setting rig max_polecats", err.Error())
			return
		}
	}

	plan.ID = types.StringValue(filepath.Join(plan.HQPath.ValueString(), plan.Name.ValueString()))
	plan.Status = types.StringValue("operational")

	prefix, err := getPrefixFromGT(ctx, runner, plan.Name.ValueString())
	if err == nil {
		plan.Prefix = types.StringValue(prefix)
	} else {
		resp.Diagnostics.AddWarning(
			"Could not determine beads prefix",
			fmt.Sprintf("Failed to read beads prefix for rig %q: %v", plan.Name.ValueString(), err),
		)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state rigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := os.Stat(state.HQPath.ValueString()); os.IsNotExist(err) {
		resp.State.RemoveResource(ctx)
		return
	}

	runner := r.runner_(state.HQPath.ValueString())

	_, err := runner.GT(ctx, "rig", "status", state.Name.ValueString())
	if err != nil {
		if tfexec.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading rig status", err.Error())
		return
	}

	// Get prefix from gt
	prefix, err := getPrefixFromGT(ctx, runner, state.Name.ValueString())
	if err == nil {
		state.Prefix = types.StringValue(prefix)
	} else {
		resp.Diagnostics.AddWarning(
			"Could not determine beads prefix",
			fmt.Sprintf("Failed to read beads prefix for rig %q: %v", state.Name.ValueString(), err),
		)
	}
	state.Status = types.StringValue("operational")

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state rigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	runner := r.runner_(plan.HQPath.ValueString())

	// Only update runtime if it actually changed
	if !plan.Runtime.Equal(state.Runtime) {
		if _, err := runner.GT(ctx, "rig", "config", "set", plan.Name.ValueString(), "runtime", plan.Runtime.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error updating rig runtime", err.Error())
			return
		}
	}

	// Only update max_polecats if it actually changed
	if !plan.MaxPolecats.Equal(state.MaxPolecats) {
		maxPolecats := fmt.Sprintf("%d", plan.MaxPolecats.ValueInt64())
		if _, err := runner.GT(ctx, "rig", "config", "set", plan.Name.ValueString(), "max_polecats", maxPolecats); err != nil {
			resp.Diagnostics.AddError("Error updating rig max_polecats", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state rigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	runner := r.runner_(state.HQPath.ValueString())

	if _, err := runner.GT(ctx, "rig", "stop", state.Name.ValueString()); err != nil {
		// Ignore errors if the rig is already gone or services are down
		if !tfexec.IsNotFound(err) {
			resp.Diagnostics.AddWarning("Error stopping rig", err.Error())
		}
	}

	if out, err := runner.GT(ctx, "rig", "dock", state.Name.ValueString()); err != nil {
		// rig dock often fails if the Dolt server was already stopped or if the
		// database is in a partial state. We treat this as a warning because
		// the resource is being removed from Terraform anyway.
		resp.Diagnostics.AddWarning(
			"Rig docked with issues",
			fmt.Sprintf("Output: %s\nError: %v", out, err),
		)
	}
}

func (r *RigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID is the rig name. HQ path comes from provider configuration.
	rigName := req.ID

	// Get HQ path from the configured runner
	hqPath := ""
	if r.runner != nil {
		hqPath = r.runner.HQPath()
	}

	if hqPath == "" {
		resp.Diagnostics.AddError(
			"Import Error",
			"Cannot import rig: hq_path must be set in the provider configuration",
		)
		return
	}

	// Set the attributes in state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), rigName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("hq_path"), hqPath)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), filepath.Join(hqPath, rigName))...)
}

func getPrefixFromGT(ctx context.Context, runner tfexec.Runner, rigName string) (string, error) {
	output, err := runner.GT(ctx, "rig", "status", rigName)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "Beads prefix:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				prefix := strings.TrimSpace(parts[1])
				// Remove trailing hyphen if present
				prefix = strings.TrimSuffix(prefix, "-")
				return prefix, nil
			}
		}
	}
	return "", fmt.Errorf("prefix not found in gt rig status output")
}
