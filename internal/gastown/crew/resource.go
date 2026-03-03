package crew

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tfexec "github.com/kybernetes-systems/terraform-provider-gastown/internal/exec"
)

var _ resource.Resource = &CrewResource{}
var _ resource.ResourceWithConfigure = &CrewResource{}

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
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"hq_path": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"rig": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role": schema.StringAttribute{
				Required: true,
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

	if _, err := runner.GT(ctx, "crew", "add", plan.Rig.ValueString(), plan.Name.ValueString(), plan.Role.ValueString()); err != nil {
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

	runner := r.runner_(state.HQPath.ValueString())

	out, err := runner.GT(ctx, "crew", "list", state.Rig.ValueString())
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

	if _, err := runner.GT(ctx, "crew", "remove", state.Rig.ValueString(), state.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error removing crew", err.Error())
		return
	}
}
