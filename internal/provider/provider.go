package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tfexec "github.com/kybernetes-systems/terraform-provider-gastown/internal/exec"
	"github.com/kybernetes-systems/terraform-provider-gastown/internal/gastown/crew"
	"github.com/kybernetes-systems/terraform-provider-gastown/internal/gastown/hq"
	"github.com/kybernetes-systems/terraform-provider-gastown/internal/gastown/rig"
)

var _ provider.Provider = &GastownProvider{}

type GastownProvider struct {
	version string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &GastownProvider{version: version}
	}
}

func (p *GastownProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "gastown"
	resp.Version = p.version
}

func (p *GastownProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"hq_path": schema.StringAttribute{
				Required:    true,
				Description: "Path to the Gas Town HQ directory.",
			},
		},
	}
}

func (p *GastownProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config struct {
		HQPath types.String `tfsdk:"hq_path"`
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if config.HQPath.IsNull() || config.HQPath.ValueString() == "" {
		resp.Diagnostics.Append(diag.NewAttributeErrorDiagnostic(
			path.Root("hq_path"),
			"Missing HQ path",
			"hq_path must be set to the Gas Town HQ directory.",
		))
		return
	}

	runner := tfexec.NewRunner(config.HQPath.ValueString())
	resp.DataSourceData = runner
	resp.ResourceData = runner
}

func (p *GastownProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		hq.New,
		rig.New,
		crew.New,
	}
}

func (p *GastownProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
