package crew

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ resource.Resource = &CrewResource{}

type CrewResource struct{}

func New() resource.Resource { return &CrewResource{} }

func (r *CrewResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_crew"
}

func (r *CrewResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
}

func (r *CrewResource) Create(_ context.Context, _ resource.CreateRequest, _ *resource.CreateResponse) {
}

func (r *CrewResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {
}

func (r *CrewResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
}

func (r *CrewResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}
