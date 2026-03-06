package rig_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestRigResource_Injection(t *testing.T) {
	fake := &fakeRunner{}
	r := newRigWithRunner(fake)
	
	// Malicious name attempting to inject extra arguments
	maliciousName := "myrig --extra-flag"
	
	attrs := baseAttrs()
	attrs["name"] = tftypes.NewValue(tftypes.String, maliciousName)
	
	plan := rigPlan(t, r, attrs)
	s := getSchema(r)
	emptyState := tfsdk.State{Raw: tftypes.NewValue(plan.Raw.Type(), nil), Schema: s.Schema}

	var resp resource.CreateResponse
	resp.State = emptyState
	
	// This currently succeeds because there's no validation
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &resp)

	if resp.Diagnostics.HasError() {
		t.Logf("Create returned errors (as expected after fix): %v", resp.Diagnostics)
	} else {
		t.Logf("Create succeeded with malicious name: %s", maliciousName)
	}
	
	// Check if the malicious name was passed as-is
	found := false
	for _, call := range fake.calls {
		for _, arg := range call {
			if arg == maliciousName {
				found = true
				break
			}
		}
	}
	
	if !found {
		t.Errorf("Expected malicious name %q to be passed as a single argument, but it wasn't found in calls: %v", maliciousName, fake.calls)
	}
}
