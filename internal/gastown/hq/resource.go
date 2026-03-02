package hq

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tfexec "github.com/kybernetes-systems/terraform-provider-gastown/internal/exec"
)

var _ resource.Resource = &HQResource{}

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
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"owner_email": schema.StringAttribute{
				Optional: true,
			},
			"git": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"name": schema.StringAttribute{
				Computed: true,
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
	Name       types.String `tfsdk:"name"`
}

func (r *HQResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan hqModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	args := []string{"install", plan.Path.ValueString()}
	if plan.Git.ValueBool() {
		args = append(args, "--git")
	}
	if !plan.OwnerEmail.IsNull() && plan.OwnerEmail.ValueString() != "" {
		args = append(args, "--owner", plan.OwnerEmail.ValueString())
	}

	runner := r.runner
	if runner == nil {
		runner = tfexec.NewRunner(plan.Path.ValueString())
	}
	if _, err := runner.GT(ctx, args...); err != nil {
		resp.Diagnostics.AddError("Error creating HQ", err.Error())
		return
	}

	name, err := readTownName(plan.Path.ValueString())
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

	townJSON := filepath.Join(state.Path.ValueString(), "mayor", "town.json")
	if _, err := os.Stat(townJSON); os.IsNotExist(err) {
		resp.State.RemoveResource(ctx)
		return
	}

	name, err := readTownName(state.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading town name", err.Error())
		return
	}
	state.Name = types.StringValue(name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *HQResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
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
	if _, err := runner.GT(ctx, "uninstall", "--force"); err != nil {
		resp.Diagnostics.AddError("Error deleting HQ", err.Error())
	}
}

func readTownName(hqPath string) (string, error) {
	data, err := os.ReadFile(filepath.Join(hqPath, "mayor", "town.json"))
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
