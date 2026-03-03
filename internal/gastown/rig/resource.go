package rig

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tfexec "github.com/kybernetes-systems/terraform-provider-gastown/internal/exec"
)

var _ resource.Resource = &RigResource{}

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
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"repo": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"runtime": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("claude"),
			},
			"max_polecats": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(3),
			},
			"status": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"prefix": schema.StringAttribute{
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

	plan.ID = types.StringValue(filepath.Join(plan.HQPath.ValueString(), plan.Name.ValueString()))
	plan.Status = types.StringValue("operational")

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state rigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	runner := r.runner_(state.HQPath.ValueString())

	out, err := runner.GT(ctx, "rig", "status", state.Name.ValueString(), "--json")
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no such") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading rig status", err.Error())
		return
	}
	_ = out

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
