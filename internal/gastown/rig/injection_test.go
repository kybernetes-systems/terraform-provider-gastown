package rig_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestRigResource_Injection(t *testing.T) {
	fake := &fakeRunner{}
	r := newRigWithRunner(fake)
	
	// Malicious name attempting to inject extra arguments
	maliciousName := "myrig --extra-flag"
	
	// Verify that the validator catches it
	s := getSchema(r)
	nameAttr := s.Schema.Attributes["name"].(schema.StringAttribute)
	
	foundValidator := false
	for _, v := range nameAttr.Validators {
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{
			Path: path.Root("name"),
			ConfigValue: types.StringValue(maliciousName),
		}, resp)
		
		if resp.Diagnostics.HasError() {
			foundValidator = true
			t.Logf("Validator correctly caught malicious name: %v", resp.Diagnostics)
		}
	}
	
	if !foundValidator {
		t.Errorf("Expected validator to catch malicious name %q, but it didn't", maliciousName)
	}

	attrs := baseAttrs()
	attrs["name"] = tftypes.NewValue(tftypes.String, maliciousName)
	
	plan := rigPlan(t, r, attrs)
	emptyState := tfsdk.State{Raw: tftypes.NewValue(plan.Raw.Type(), nil), Schema: s.Schema}

	var resp resource.CreateResponse
	resp.State = emptyState
	
	// We still call Create to ensure it's at least not crashing, 
	// though in real TF validation would have stopped it before Create.
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &resp)
}
