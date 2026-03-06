package provider_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const testRepoURL = "https://github.com/google/googletest.git"

func TestAcc_FullLifecycle(t *testing.T) {
	hqPath := filepath.Join(t.TempDir(), "gt-lifecycle")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConfig(hqPath, "mirror", "claude", "deacon", testRepoURL),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gastown_hq.main", "path", hqPath),
					resource.TestCheckResourceAttr("gastown_rig.mirror", "name", "mirror"),
					resource.TestCheckResourceAttr("gastown_rig.mirror", "runtime", "claude"),
					resource.TestCheckResourceAttr("gastown_crew.deacon", "name", "deacon"),
				),
			},
			{
				Config:             testAccConfig(hqPath, "mirror", "claude", "deacon", testRepoURL),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: testAccConfig(hqPath, "mirror", "gemini", "deacon", testRepoURL),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gastown_rig.mirror", "runtime", "gemini"),
				),
			},
		},
	})
}

func TestAcc_DriftScenario(t *testing.T) {
	hqPath := filepath.Join(t.TempDir(), "gt-drift")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConfig(hqPath, "drift_rig", "claude", "drift_crew", testRepoURL),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						// Manually edit rig config.json to cause drift
						configPath := filepath.Join(hqPath, "drift_rig", "config.json")
						content, err := os.ReadFile(configPath)
						if err != nil {
							return err
						}
						newContent := strings.Replace(string(content), `"runtime": "claude"`, `"runtime": "gemini"`, 1)
						return os.WriteFile(configPath, []byte(newContent), 0644)
					},
				),
			},
			{
				Config:             testAccConfig(hqPath, "drift_rig", "claude", "drift_crew", testRepoURL),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAcc_Concurrency(t *testing.T) {
	t.Parallel()

	hqPath1 := filepath.Join(t.TempDir(), "gt-con-1")
	hqPath2 := filepath.Join(t.TempDir(), "gt-con-2")

	// Start two Test functions in parallel
	t.Run("first", func(t *testing.T) {
		t.Parallel()
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: TestAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccConfig(hqPath1, "rig1", "claude", "crew1", testRepoURL),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("gastown_rig.mirror", "name", "rig1"),
					),
				},
			},
		})
	})

	t.Run("second", func(t *testing.T) {
		t.Parallel()
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: TestAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccConfig(hqPath2, "rig2", "claude", "crew2", testRepoURL),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("gastown_rig.mirror", "name", "rig2"),
					),
				},
			},
		})
	})
}

func testAccConfig(hqPath, rigName, runtime, crewName, repoURL string) string {
	return fmt.Sprintf(`
provider "gastown" {
  hq_path = %[1]q
}

resource "gastown_hq" "main" {
  path        = %[1]q
  owner_email = "test@example.com"
}

resource "gastown_rig" "mirror" {
  hq_path      = gastown_hq.main.path
  name         = %[2]q
  repo         = %[5]q
  runtime      = %[3]q
  max_polecats = 0
}

resource "gastown_crew" "deacon" {
  hq_path  = gastown_hq.main.path
  rig      = gastown_rig.mirror.name
  name     = %[4]q
  role     = "operator"
}
`, hqPath, rigName, runtime, crewName, repoURL)
}
