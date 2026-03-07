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
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the rig resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"hq_path": schema.StringAttribute{
				Description: "Path to the Gas Town HQ directory.",
				Required:    true,
				Validators: []validator.String{
					validators.PathValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the rig (used as identifier).",
				Required:    true,
				Validators: []validator.String{
					validators.SafeNameValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"repo": schema.StringAttribute{
				Description: "Git repository URL or local path for the rig.",
				Required:    true,
				Validators: []validator.String{
					validators.RepoURLValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"runtime": schema.StringAttribute{
				Description: "Runtime environment for the rig. Defaults to 'claude'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("claude"),
				Validators: []validator.String{
					validators.SafeNameValidator{},
				},
			},
			"max_polecats": schema.Int64Attribute{
				Description: "Maximum number of polecats (workers) for the rig. Defaults to 3.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(3),
			},
			"status": schema.StringAttribute{
				Description: "Current operational status of the rig.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"prefix": schema.StringAttribute{
				Description: "Beads prefix assigned to this rig (read from gt rig status).",
				Computed:    true,
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
	}
	state.Status = types.StringValue("operational")

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan rigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	runner := r.runner_(plan.HQPath.ValueString())

	if _, err := runner.GT(ctx, "rig", "config", "set", plan.Name.ValueString(), "runtime", plan.Runtime.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error updating rig runtime", err.Error())
		return
	}

	// Update max_polecats if changed
	if !plan.MaxPolecats.IsNull() && !plan.MaxPolecats.IsUnknown() {
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
		resp.Diagnostics.AddError("Error stopping rig", err.Error())
		return
	}

	if _, err := runner.GT(ctx, "rig", "dock", state.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error docking rig", err.Error())
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
