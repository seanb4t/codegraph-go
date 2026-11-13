package upgrade

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	yaml "go.yaml.in/yaml/v3"
)

// sbomAttestPlatforms are the release platforms. Each one ships one binary
// and one SBOM, so each one needs one SBOM attestation.
var sbomAttestPlatforms = []string{"linux_amd64", "linux_arm64", "darwin_amd64", "darwin_arm64"}

var sbomAttestPinnedRe = regexp.MustCompile(`^actions/attest@[0-9a-f]{40}$`)

type sbomAttestStep struct {
	ID   string            `yaml:"id"`
	Uses string            `yaml:"uses"`
	Run  string            `yaml:"run"`
	With map[string]string `yaml:"with"`
}

type sbomAttestWorkflow struct {
	Jobs map[string]struct {
		Steps []sbomAttestStep `yaml:"steps"`
	} `yaml:"jobs"`
}

type sbomAttestTaskfile struct {
	Tasks map[string]struct {
		Cmds []any `yaml:"cmds"`
	} `yaml:"tasks"`
}

// TestSBOMAttestStepsBindOneSBOMToOneBinary guards the #294 shape of
// release.yml. The job that runs GoReleaser must attest one SBOM per
// platform, and each step must name the binary, the digest output and the
// SBOM of the same platform. A step that takes the checksums file instead
// binds one SBOM to all 8 subjects, so that input is rejected.
func TestSBOMAttestStepsBindOneSBOMToOneBinary(t *testing.T) {
	releaseData, err := os.ReadFile(releaseWorkflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", releaseWorkflowPath, err)
	}
	goreleaserJob, err := parseGoreleaserInvokingJob(string(releaseData))
	if err != nil {
		t.Fatalf("parseGoreleaserInvokingJob: %v", err)
	}
	var wf sbomAttestWorkflow
	if err := yaml.Unmarshal(releaseData, &wf); err != nil {
		t.Fatalf("yaml.Unmarshal(%s): %v", releaseWorkflowPath, err)
	}

	const subjectsTask = "release:sbom-subjects"
	subjectsStepID := ""
	subjectsStepIndex := -1
	var attested []string
	for jobID, job := range wf.Jobs {
		for i, step := range job.Steps {
			if strings.TrimSpace(step.Run) == "task "+subjectsTask {
				if jobID != goreleaserJob.ID {
					t.Errorf("the %s step lives in job %q, want %q", subjectsTask, jobID, goreleaserJob.ID)
				}
				subjectsStepID, subjectsStepIndex = step.ID, i
			}
			if !strings.HasPrefix(step.Uses, "actions/attest@") {
				continue
			}
			if jobID != goreleaserJob.ID {
				t.Errorf("an actions/attest step lives in job %q, want %q (one job holds the signing identity)", jobID, goreleaserJob.ID)
			}
			if !sbomAttestPinnedRe.MatchString(step.Uses) {
				t.Errorf("uses: %q, want actions/attest pinned to a full 40-hex commit SHA", step.Uses)
			}
			if _, ok := step.With["subject-checksums"]; ok {
				t.Errorf("step %q passes subject-checksums, which binds one SBOM to every subject in the file", step.With["subject-name"])
			}
			platform, ok := strings.CutPrefix(step.With["subject-name"], "codegraph_${{ github.ref_name }}_")
			if !ok {
				t.Errorf("subject-name: %q, want codegraph_${{ github.ref_name }}_<goos>_<goarch>", step.With["subject-name"])
				continue
			}
			attested = append(attested, platform)
			if subjectsStepIndex == -1 || i < subjectsStepIndex {
				t.Errorf("the SBOM attest step for %s runs before the %s step", platform, subjectsTask)
			}
			wantDigest := fmt.Sprintf("${{ steps.%s.outputs.%s }}", subjectsStepID, platform)
			if got := step.With["subject-digest"]; got != wantDigest {
				t.Errorf("%s: subject-digest: %q, want %q", platform, got, wantDigest)
			}
			wantSBOM := fmt.Sprintf("./dist/codegraph_${{ github.ref_name }}_%s.spdx.json", platform)
			if got := step.With["sbom-path"]; got != wantSBOM {
				t.Errorf("%s: sbom-path: %q, want %q", platform, got, wantSBOM)
			}
		}
	}
	if subjectsStepID == "" {
		t.Fatalf("no step with an id runs `task %s`", subjectsTask)
	}

	want := slices.Sorted(slices.Values(sbomAttestPlatforms))
	slices.Sort(attested)
	if !slices.Equal(attested, want) {
		t.Errorf("SBOM attest steps cover %v, want exactly %v", attested, want)
	}

	// The workflow reads one output per platform. The task must write each one.
	taskfileData, err := os.ReadFile(taskfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", taskfilePath, err)
	}
	var tf sbomAttestTaskfile
	if err := yaml.Unmarshal(taskfileData, &tf); err != nil {
		t.Fatalf("yaml.Unmarshal(%s): %v", taskfilePath, err)
	}
	body := fmt.Sprint(tf.Tasks[subjectsTask].Cmds...)
	wantLoop := "for suf in " + strings.Join(sbomAttestPlatforms, " ") + "; do"
	if !strings.Contains(body, wantLoop) {
		t.Errorf("task %s does not loop over the platforms: want %q in its commands", subjectsTask, wantLoop)
	}
}
