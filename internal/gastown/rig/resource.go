package rig

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ resource.Resource = &RigResource{}

type RigResource struct{}

func New() resource.Resource { return &RigResource{} }

func (r *RigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rig"
}

func (r *RigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
}

func (r *RigResource) Create(_ context.Context, _ resource.CreateRequest, _ *resource.CreateResponse) {
}

func (r *RigResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {
}

func (r *RigResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
}

func (r *RigResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}
