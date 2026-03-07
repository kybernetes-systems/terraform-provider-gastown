package provider_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	tfexec "github.com/kybernetes-systems/terraform-provider-gastown/internal/exec"
	ourprovider "github.com/kybernetes-systems/terraform-provider-gastown/internal/provider"
	"github.com/kybernetes-systems/terraform-provider-gastown/internal/testutil"
)

var (
	TestAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"gastown": providerserver.NewProtocol6WithError(ourprovider.New("test")()),
	}
	TestAccFakeProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"gastown": providerserver.NewProtocol6WithError(ourprovider.NewForTesting("test", testutil.NewFakeRunner)()),
	}
)

func newProvider(t *testing.T) provider.Provider {
	t.Helper()
	return ourprovider.New("test")()
}

// Test 1: schema exposes hq_path as a required string attribute.
func TestProvider_schema_hq_path(t *testing.T) {
	p := newProvider(t)
	var resp provider.SchemaResponse
	p.Schema(context.Background(), provider.SchemaRequest{}, &resp)

	attr, ok := resp.Schema.Attributes["hq_path"]
	if !ok {
		t.Fatal("provider schema missing hq_path attribute")
	}
	sa, ok := attr.(schema.StringAttribute)
	if !ok {
		t.Fatalf("hq_path should be a StringAttribute, got %T", attr)
	}
	if !sa.Required {
		t.Fatal("hq_path should be Required")
	}
}

// Test 2: Configure adds an error diagnostic when hq_path is empty.
func TestProvider_configure_rejects_empty_hq_path(t *testing.T) {
	p := newProvider(t)

	var schemaResp provider.SchemaResponse
	p.Schema(context.Background(), provider.SchemaRequest{}, &schemaResp)

	configVal := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{"hq_path": tftypes.String}},
		map[string]tftypes.Value{"hq_path": tftypes.NewValue(tftypes.String, "")},
	)
	req := provider.ConfigureRequest{
		Config: tfsdk.Config{Raw: configVal, Schema: schemaResp.Schema},
	}
	var resp provider.ConfigureResponse
	p.Configure(context.Background(), req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error diagnostic for empty hq_path, got none")
	}
}

// Test 3: Resources returns constructors for all three resource types.
func TestProvider_registers_three_resources(t *testing.T) {
	p := newProvider(t)
	resources := p.Resources(context.Background())
	if len(resources) != 3 {
		t.Fatalf("expected 3 registered resources, got %d", len(resources))
	}
}

func TestHQ_Schema_NoBeads(t *testing.T) {
	p := newProvider(t)
	resources := p.Resources(context.Background())
	var hqRes resource.Resource
	for _, rFunc := range resources {
		r := rFunc()
		var metadataResp resource.MetadataResponse
		r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "gastown"}, &metadataResp)
		if metadataResp.TypeName == "gastown_hq" {
			hqRes = r
			break
		}
	}

	if hqRes == nil {
		t.Fatal("gastown_hq resource not found")
	}

	var schemaResp resource.SchemaResponse
	hqRes.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	if _, ok := schemaResp.Schema.Attributes["no_beads"]; !ok {
		t.Fatal("gastown_hq schema missing no_beads attribute")
	}
}

// TestProvider_NewForTesting verifies NewForTesting factory creates provider with injected runner.
func TestProvider_NewForTesting(t *testing.T) {
	factory := ourprovider.NewForTesting("test", testutil.NewFakeRunner)
	p := factory()

	var schemaResp provider.SchemaResponse
	p.Schema(context.Background(), provider.SchemaRequest{}, &schemaResp)

	configVal := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{"hq_path": tftypes.String}},
		map[string]tftypes.Value{"hq_path": tftypes.NewValue(tftypes.String, "/tmp/testhq")},
	)
	req := provider.ConfigureRequest{
		Config: tfsdk.Config{Raw: configVal, Schema: schemaResp.Schema},
	}
	var resp provider.ConfigureResponse
	p.Configure(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// ResourceData should be a FakeRunner
	_, ok := resp.ResourceData.(tfexec.Runner)
	if !ok {
		t.Errorf("expected ResourceData to implement Runner interface, got %T", resp.ResourceData)
	}
}
