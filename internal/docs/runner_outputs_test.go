package docs

import (
	"fmt"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func releaseRunnerContract(body string) error {
	var document struct {
		Permissions map[string]string `yaml:"permissions"`
		Jobs        map[string]struct {
			RunsOn string `yaml:"runs-on"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal([]byte(body), &document); err != nil {
		return err
	}
	base := "${{ (needs.setup.outputs.runs-on == 'ubuntu-latest' && 'ubuntu-latest') || 'blacksmith-4vcpu-ubuntu-2404' }}"
	want := map[string]string{"binaries": base, "images": base, "runner-manifests": base,
		"runner-images": "${{ matrix.arch == 'arm64' && 'ubuntu-24.04-arm' || (needs.setup.outputs.runs-on == 'ubuntu-latest' && 'ubuntu-latest') || 'blacksmith-4vcpu-ubuntu-2404' }}"}
	for name, expected := range want {
		if document.Jobs[name].RunsOn != expected {
			return fmt.Errorf("job %s runner differs: %s", name, document.Jobs[name].RunsOn)
		}
	}
	if len(document.Permissions) != 1 || document.Permissions["contents"] != "read" {
		return fmt.Errorf("release default permissions must remain contents: read")
	}
	return nil
}

func TestReleaseRunnerOutputsAreClosed(t *testing.T) {
	if err := releaseRunnerContract(workflowFiles(t)["release.yml"]); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseRunnerContractAllowsActionPinMaintenance(t *testing.T) {
	body := workflowFiles(t)["release.yml"]
	for _, match := range usesLine.FindAllStringSubmatch(body, -1) {
		ref := match[1]
		name, sha, found := strings.Cut(ref, "@")
		if !found || len(sha) != 40 {
			continue
		}
		replacement := strings.Repeat("a", 40)
		if sha == replacement {
			replacement = strings.Repeat("f", 40)
		}
		updated := strings.Replace(body, ref, name+"@"+replacement, 1)
		if err := releaseRunnerContract(updated); err != nil {
			t.Fatalf("action-pin-only maintenance changed runner contract: %v", err)
		}
		return
	}
	t.Fatal("release workflow has no pinned action for maintenance control")
}
