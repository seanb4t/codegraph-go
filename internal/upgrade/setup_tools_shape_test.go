package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// setupToolsActionYAML is the subset of .github/actions/setup-tools/action.yml
// the shape guard reads.
type setupToolsActionYAML struct {
	Inputs map[string]struct {
		Default string `yaml:"default"`
	} `yaml:"inputs"`
	Runs struct {
		Steps []compositeCacheStep `yaml:"steps"`
	} `yaml:"runs"`
}

// remotePinnedUsesRe is the only accepted shape for a remote uses: line in a
// local composite action: owner/repo@<40 hex> # vX.Y.Z.
var remotePinnedUsesRe = regexp.MustCompile(`^\s*(?:-\s+)?uses:\s*[A-Za-z0-9._-]+/[A-Za-z0-9._/-]+@[0-9a-f]{40} # v[0-9]+\.[0-9]+\.[0-9]+\s*$`)

// TestSetupToolsActionShape pins the setup-tools action's own contract, which
// every caller-side guard relies on: the input defaults, the mise-action
// inputs that carry them through, the go-cache gate on both Go cache steps,
// and a full SHA pin on every remote action it uses.
func TestSetupToolsActionShape(t *testing.T) {
	src, err := os.ReadFile(setupToolsActionPath)
	if err != nil {
		t.Fatalf("read %s: %v", setupToolsActionPath, err)
	}
	var action setupToolsActionYAML
	if err := yaml.Unmarshal(src, &action); err != nil {
		t.Fatalf("yaml.Unmarshal(%s): %v", setupToolsActionPath, err)
	}

	wantDefaults := map[string]string{"tools": "", "cache": "true", "go-cache": "false"}
	for name, want := range wantDefaults {
		in, ok := action.Inputs[name]
		if !ok {
			t.Errorf("%s declares no %q input", setupToolsActionPath, name)
			continue
		}
		if in.Default != want {
			t.Errorf("%s input %q defaults to %q, want %q", setupToolsActionPath, name, in.Default, want)
		}
	}

	var miseSteps, gatedSteps int
	for _, step := range action.Runs.Steps {
		if actionPrefix(step.Uses) == "jdx/mise-action" {
			miseSteps++
			want := map[string]string{
				"install_args":      "${{ inputs.tools }}",
				"cache":             "${{ inputs.cache }}",
				"add_shims_to_path": "false",
			}
			for k, v := range want {
				if got := fmt.Sprint(step.With[k]); got != v {
					t.Errorf("%s mise-action step with.%s = %q, want %q", setupToolsActionPath, k, got, v)
				}
			}
		}
		if strings.TrimSpace(step.If) == goCacheGate {
			gatedSteps++
		}
	}
	if miseSteps != 1 {
		t.Errorf("%s has %d jdx/mise-action steps, want exactly 1", setupToolsActionPath, miseSteps)
	}
	if gatedSteps != 2 {
		t.Errorf("%s has %d steps gated by %q, want 2 (locate and cache the Go caches)", setupToolsActionPath, gatedSteps, goCacheGate)
	}

	pins := 0
	for line := range strings.SplitSeq(string(src), "\n") {
		if !strings.Contains(line, "uses:") || strings.Contains(line, "uses: ./") {
			continue
		}
		if !remotePinnedUsesRe.MatchString(line) {
			t.Errorf("%s: %q is not a full 40-hex SHA pin with a ` # vX.Y.Z` comment", setupToolsActionPath, strings.TrimSpace(line))
			continue
		}
		pins++
	}
	if pins < 2 {
		t.Errorf("%s: checked %d remote uses: pins, want at least 2 (mise-action and cache)", setupToolsActionPath, pins)
	}
	t.Logf("%s: %d inputs, %d mise-action step, %d go-cache-gated steps, %d SHA pins", setupToolsActionPath, len(action.Inputs), miseSteps, gatedSteps, pins)
}

func TestSetupToolsList(t *testing.T) {
	cases := []struct {
		name  string
		with  map[string]any
		tools []string
		all   bool
	}{
		{"no with", nil, nil, true},
		{"tools absent", map[string]any{"cache": "false"}, nil, true},
		{"tools null", map[string]any{"tools": nil}, nil, true},
		{"tools blank", map[string]any{"tools": "  "}, []string{}, true},
		{"tools listed", map[string]any{"tools": "go task"}, []string{"go", "task"}, false},
	}
	for _, c := range cases {
		tools, all := setupToolsList(c.with)
		if all != c.all || len(tools) != len(c.tools) || (len(tools) > 0 && !slices.Equal(tools, c.tools)) {
			t.Errorf("%s: setupToolsList = %v, %v; want %v, %v", c.name, tools, all, c.tools, c.all)
		}
	}

	miseToml := "[tools]\ncosign = \"3.1.4\"\n"
	step := func(with map[string]any) workflowRunStep { return workflowRunStep{Uses: setupToolsAction, With: with} }
	installs := []struct {
		name string
		step workflowRunStep
		toml string
		want bool
	}{
		{"every tool, cosign pinned", step(nil), miseToml, true},
		{"every tool, cosign not pinned", step(nil), "[tools]\ngo = \"1.26.10\"\n", false},
		{"list names cosign", step(map[string]any{"tools": "go cosign"}), miseToml, true},
		{"list omits cosign", step(map[string]any{"tools": "go task"}), miseToml, false},
		{"another action", workflowRunStep{Uses: "sigstore/cosign-installer@v4"}, miseToml, false},
	}
	for _, c := range installs {
		if got := setupToolsInstalls(c.step, "cosign", c.toml); got != c.want {
			t.Errorf("%s: setupToolsInstalls = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestPullRequestTargetJobsUseNoToolCache holds every setup-tools step in a
// pull_request_target workflow to `cache: "false"` and an explicit tools
// list: those jobs run on fork pull requests and must neither read a mutable
// cache nor install tools they do not run.
func TestPullRequestTargetJobsUseNoToolCache(t *testing.T) {
	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		t.Fatalf("read %s: %v", workflowsDir, err)
	}
	checked := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yml") {
			continue
		}
		path := filepath.Join(workflowsDir, e.Name())
		doc, err := decodeFullWorkflowDoc(path)
		if err != nil {
			t.Fatalf("decodeFullWorkflowDoc(%s): %v", path, err)
		}
		if !workflowOnTriggers(doc.On)["pull_request_target"] {
			continue
		}
		for jobID, job := range doc.Jobs {
			for _, step := range job.Steps {
				if step.Uses != setupToolsAction {
					continue
				}
				checked++
				if v := fmt.Sprint(step.With["cache"]); v != "false" {
					t.Errorf("%s job %q step %q has cache = %q, want \"false\" under pull_request_target", e.Name(), jobID, step.Name, v)
				}
				if _, all := setupToolsList(step.With); all {
					t.Errorf("%s job %q step %q installs every mise tool; list only what the job runs", e.Name(), jobID, step.Name)
				}
			}
		}
	}
	if checked == 0 {
		t.Fatalf("no setup-tools step found in any pull_request_target workflow — this guard would pass vacuously")
	}
	t.Logf("checked %d setup-tools steps in pull_request_target workflows", checked)
}

// TestProtocGenGoMatchesProtobufRuntime holds mise.toml's protoc-gen-go pin
// equal to go.mod's google.golang.org/protobuf requirement: generated code
// must match the runtime it links against.
func TestProtocGenGoMatchesProtobufRuntime(t *testing.T) {
	miseData, err := os.ReadFile(miseTomlPath)
	if err != nil {
		t.Fatalf("read %s: %v", miseTomlPath, err)
	}
	pin, err := parseMiseToolVersion(string(miseData), "protoc-gen-go")
	if err != nil {
		t.Fatalf("%s: %v", miseTomlPath, err)
	}
	goModData, err := os.ReadFile(rootGoModPath)
	if err != nil {
		t.Fatalf("read %s: %v", rootGoModPath, err)
	}
	runtime, err := parseGoModRequireVersion(string(goModData), "google.golang.org/protobuf")
	if err != nil {
		t.Fatalf("%s: %v", rootGoModPath, err)
	}
	if strings.TrimPrefix(runtime, "v") != strings.TrimPrefix(pin, "v") {
		t.Fatalf("mise.toml pins protoc-gen-go %s but go.mod requires google.golang.org/protobuf %s — they must match", pin, runtime)
	}
}
