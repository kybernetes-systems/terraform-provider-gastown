// Package validators provides custom Terraform validators for the Gas Town provider.
package validators

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// PathValidator validates that a path is safe (no traversal, absolute, no shell metacharacters).
type PathValidator struct{}

func (v PathValidator) Description(ctx context.Context) string {
	return "path must be absolute, contain no parent directory references (..), and no shell metacharacters"
}

func (v PathValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v PathValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	path := req.ConfigValue.ValueString()

	// Must be absolute path
	if !filepath.IsAbs(path) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Path",
			fmt.Sprintf("Path must be absolute, got: %s", path),
		)
		return
	}

	// No parent directory references
	if strings.Contains(path, "..") {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Path",
			fmt.Sprintf("Path cannot contain parent directory references (..): %s", path),
		)
		return
	}

	// No shell metacharacters
	shellChars := []string{"|", "&", ";", "$", "`", "\\", "*", "?", "<", ">", "("}
	for _, char := range shellChars {
		if strings.Contains(path, char) {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid Path",
				fmt.Sprintf("Path contains invalid character %q: %s", char, path),
			)
			return
		}
	}
}

// SafeNameValidator validates that a name contains only safe characters.
type SafeNameValidator struct{}

func (v SafeNameValidator) Description(ctx context.Context) string {
	return "name must contain only alphanumeric characters, hyphens, and underscores"
}

func (v SafeNameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

var safeNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_-]*$`)

func (v SafeNameValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	name := req.ConfigValue.ValueString()

	if !safeNamePattern.MatchString(name) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Name",
			fmt.Sprintf("Name must contain only alphanumeric characters, hyphens, and underscores: %s", name),
		)
	}
}

// EmailValidator validates basic email format.
type EmailValidator struct{}

func (v EmailValidator) Description(ctx context.Context) string {
	return "must be a valid email address"
}

func (v EmailValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

var emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func (v EmailValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	email := req.ConfigValue.ValueString()

	if !emailPattern.MatchString(email) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Email",
			fmt.Sprintf("Must be a valid email address: %s", email),
		)
	}
}

// RepoURLValidator validates that a repo URL is safe.
type RepoURLValidator struct{}

func (v RepoURLValidator) Description(ctx context.Context) string {
	return "repository URL must be a valid git URL or local path"
}

func (v RepoURLValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v RepoURLValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	repo := req.ConfigValue.ValueString()

	// Check for shell metacharacters
	shellChars := []string{"|", "&", ";", "`", "$", "\n", "\r"}
	for _, char := range shellChars {
		if strings.Contains(repo, char) {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid Repository URL",
				fmt.Sprintf("Repository URL contains invalid character %q: %s", char, repo),
			)
			return
		}
	}

	// Basic validation - must look like a URL or path
	if repo == "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Repository URL",
			"Repository URL cannot be empty",
		)
		return
	}
}

// RoleValidator validates crew role names.
type RoleValidator struct{}

func (v RoleValidator) Description(ctx context.Context) string {
	return "role must contain only alphanumeric characters, hyphens, and underscores"
}

func (v RoleValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v RoleValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	role := req.ConfigValue.ValueString()

	if !safeNamePattern.MatchString(role) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Role",
			fmt.Sprintf("Role must contain only alphanumeric characters, hyphens, and underscores: %s", role),
		)
	}
}
