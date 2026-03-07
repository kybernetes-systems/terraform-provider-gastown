package validators

import (
	"context"
	"runtime"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPathValidator(t *testing.T) {
	v := PathValidator{}
	ctx := context.Background()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid absolute path", "/tmp/gt", false},
		{"relative path", "gt", true},
		{"parent directory", "/tmp/../etc", true},
		{"shell injection", "/tmp/gt; rm -rf /", true},
		{"backticks", "/tmp/`whoami`", true},
		{"variable injection", "/tmp/$HOME", true},
	}

	if runtime.GOOS == "windows" {
		tests = append(tests, struct {
			name    string
			path    string
			wantErr bool
		}{"valid windows path", "C:\\gt", false})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			v.ValidateString(ctx, validator.StringRequest{
				Path:        path.Root("path"),
				ConfigValue: types.StringValue(tt.path),
			}, resp)

			if (resp.Diagnostics.HasError()) != tt.wantErr {
				t.Errorf("PathValidator.ValidateString(%q) error = %v, wantErr %v", tt.path, resp.Diagnostics.Errors(), tt.wantErr)
			}
		})
	}
}

func TestSafeNameValidator(t *testing.T) {
	v := SafeNameValidator{}
	ctx := context.Background()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid name", "my-rig_1", false},
		{"with spaces", "my rig", true},
		{"with dots", "my.rig", true},
		{"with slash", "my/rig", true},
		{"flag injection", "--flag", true},
		{"command injection", "rig; rm", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			v.ValidateString(ctx, validator.StringRequest{
				Path:        path.Root("name"),
				ConfigValue: types.StringValue(tt.input),
			}, resp)

			if (resp.Diagnostics.HasError()) != tt.wantErr {
				t.Errorf("SafeNameValidator.ValidateString(%q) error = %v, wantErr %v", tt.input, resp.Diagnostics.Errors(), tt.wantErr)
			}
		})
	}
}

func TestRepoURLValidator(t *testing.T) {
	v := RepoURLValidator{}
	ctx := context.Background()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid https", "https://github.com/org/repo.git", false},
		{"valid ssh", "git@github.com:org/repo.git", false},
		{"valid local", "/tmp/repo", false},
		{"shell injection", "https://repo.git; rm -rf /", true},
		{"newlines", "https://repo.git\nrm -rf /", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			v.ValidateString(ctx, validator.StringRequest{
				Path:        path.Root("repo"),
				ConfigValue: types.StringValue(tt.input),
			}, resp)

			if (resp.Diagnostics.HasError()) != tt.wantErr {
				t.Errorf("RepoURLValidator.ValidateString(%q) error = %v, wantErr %v", tt.input, resp.Diagnostics.Errors(), tt.wantErr)
			}
		})
	}
}
