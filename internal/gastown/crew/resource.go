package crew

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tfexec "github.com/kybernetes-systems/terraform-provider-gastown/internal/exec"
	"github.com/kybernetes-systems/terraform-provider-gastown/internal/validators"
)

var _ resource.Resource = &CrewResource{}
var _ resource.ResourceWithConfigure = &CrewResource{}
var _ resource.ResourceWithImportState = &CrewResource{}

type CrewResource struct {
	runner tfexec.Runner
}

func New() resource.Resource { return &CrewResource{} }

func (r *CrewResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CrewResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_crew"
}

func (r *CrewResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a crew member assignment within a Gas Town rig. Crew members represent " +
			"individual agents or workers that operate within a rig's context, each with a specific functional role. " +
			"A crew member's identity is tied to both the rig and the role they perform—changing either requires " +
			"replacement of the crew resource. Crew assignments are stored in the HQ's Dolt database and synced " +
			"across all workspace clones.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the crew resource, constructed as the join of hq_path, rig name, " +
					"and crew member name. This computed value remains stable throughout the resource lifecycle " +
					"and is used for internal resource tracking.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"hq_path": schema.StringAttribute{
				Description: "Absolute path to the Gas Town HQ directory where this crew member will be registered. " +
					"The crew is associated with this specific HQ instance for operational and state tracking purposes. " +
					"Changing this value after creation forces replacement of the crew member—the existing assignment " +
					"is removed and a new one is created in the specified HQ.",
				Required: true,
				Validators: []validator.String{
					validators.PathValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"rig": schema.StringAttribute{
				Description: "Name of the rig to which this crew member is assigned. The rig determines the " +
					"execution context, available resources, and operational scope for this crew member. The crew " +
					"member operates within the rig's configured runtime environment and respects the rig's " +
					"max_polecats limits. Changing this value after creation forces replacement of the crew member. " +
					"Must be a valid safe name containing only alphanumeric characters, hyphens, and underscores.",
				Required: true,
				Validators: []validator.String{
					validators.SafeNameValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Unique name for this crew member within the rig. The name serves as the primary " +
					"identifier for crew operations and is used in logs, status output, and operational commands. " +
					"Set once at creation; changing this value forces replacement of the crew member. " +
					"Corresponds to the <name> argument of `gt crew add`. " +
					"Must be a valid safe name containing only alphanumeric characters, hyphens, and underscores.",
				Required: true,
				Validators: []validator.String{
					validators.SafeNameValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role": schema.StringAttribute{
				Description: "Functional role assigned to this crew member, determining its capabilities and " +
					"default behaviors. Standard roles include 'coder' for development tasks and 'reviewer' for " +
					"code review operations. Each role has a specific implementation within the rig's repository " +
					"that defines how the crew member processes tasks. Changing this value after creation forces " +
					"replacement of the crew member to ensure proper role initialization.",
				Required: true,
				Validators: []validator.String{
					validators.RoleValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

type crewModel struct {
	ID     types.String `tfsdk:"id"`
	HQPath types.String `tfsdk:"hq_path"`
	Rig    types.String `tfsdk:"rig"`
	Name   types.String `tfsdk:"name"`
	Role   types.String `tfsdk:"role"`
}

func (r *CrewResource) runner_(hqPath string) tfexec.Runner {
	if r.runner != nil {
		return r.runner
	}
	return tfexec.NewRunner(hqPath)
}

func (r *CrewResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan crewModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	runner := r.runner_(plan.HQPath.ValueString())

	if _, err := runner.GT(ctx, "crew", "add", "--rig", plan.Rig.ValueString(), plan.Name.ValueString(), plan.Role.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error creating crew", err.Error())
		return
	}

	plan.ID = types.StringValue(filepath.Join(plan.HQPath.ValueString(), plan.Rig.ValueString(), plan.Name.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CrewResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state crewModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := os.Stat(state.HQPath.ValueString()); os.IsNotExist(err) {
		resp.Diagnostics.AddWarning(
			"HQ path not found",
			fmt.Sprintf("HQ directory %q does not exist. Removing crew resource from state.", state.HQPath.ValueString()),
		)
		resp.State.RemoveResource(ctx)
		return
	}

	runner := r.runner_(state.HQPath.ValueString())

	out, err := runner.GT(ctx, "crew", "list", "--rig", state.Rig.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading crew", err.Error())
		return
	}

	found := false
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if strings.Contains(line, state.Name.ValueString()) {
			found = true
			break
		}
	}

	if !found {
		resp.Diagnostics.AddWarning(
			"Crew member not found",
			fmt.Sprintf("Crew member %q not found in rig %q. Removing from state.", state.Name.ValueString(), state.Rig.ValueString()),
		)
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CrewResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Crew resources must be replaced for any change.")
}

func (r *CrewResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state crewModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	runner := r.runner_(state.HQPath.ValueString())

	if _, err := runner.GT(ctx, "crew", "remove", "--rig", state.Rig.ValueString(), "--force", state.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error removing crew", err.Error())
		return
	}
}

func (r *CrewResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID is <rig_name>/<crew_name>. HQ path comes from provider configuration.
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Import Error",
			fmt.Sprintf("Invalid import ID %q: expected format <rig_name>/<crew_name>", req.ID),
		)
		return
	}
	rigName := parts[0]
	crewName := parts[1]

	// Get HQ path from the configured runner
	hqPath := ""
	if r.runner != nil {
		hqPath = r.runner.HQPath()
	}

	if hqPath == "" {
		resp.Diagnostics.AddError(
			"Import Error",
			"Cannot import crew: hq_path must be set in the provider configuration",
		)
		return
	}

	// Set the attributes in state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("rig"), rigName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), crewName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("hq_path"), hqPath)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), filepath.Join(hqPath, rigName, crewName))...)
}
