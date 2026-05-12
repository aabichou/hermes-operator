package v1

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Ptr is a local test helper (the resources package's Ptr is not visible here).
func Ptr[T any](v T) *T { return &v }

// TestHermesInstanceSpec_HasAllSubSpecs is the schema canary — every sub-spec
// from design §4 must be addressable on HermesInstanceSpec. Tasks 3-9 fill the
// bodies; this test only guards the shape so the field-tag / json-name choices
// are reviewable in one place.
func TestHermesInstanceSpec_HasAllSubSpecs(t *testing.T) {
	t.Parallel()

	specType := reflect.TypeOf(HermesInstanceSpec{})
	required := []string{
		"Image", "Config", "Workspace", "Resources", "Security", "Storage",
		"Networking", "Observability", "Availability", "Probes",
		"Scheduling", "InitContainers", "Sidecars", "ExtraVolumes",
		"ExtraVolumeMounts", "EnvFrom", "Env", "Skills",
		"SelfConfigure", "Suspended",
	}
	for _, name := range required {
		_, ok := specType.FieldByName(name)
		assert.Truef(t, ok, "HermesInstanceSpec is missing field %q (design §4)", name)
	}
}

func TestConfigSpec_RawAndRef(t *testing.T) {
	t.Parallel()
	cs := ConfigSpec{
		Raw:          &RawConfig{RawExtension: runtime.RawExtension{Raw: []byte(`{"a":1}`)}},
		ConfigMapRef: &corev1.LocalObjectReference{Name: "user-config"},
		MergeMode:    ConfigMergeModeMerge,
	}
	assert.NotNil(t, cs.Raw)
	assert.NotNil(t, cs.ConfigMapRef)
	assert.Equal(t, ConfigMergeModeMerge, cs.MergeMode)
}

func TestWorkspaceSpec_NestedPath(t *testing.T) {
	t.Parallel()
	ws := WorkspaceSpec{
		InitialFiles: []WorkspaceFile{
			{Path: "notes/finance/2026.md", Content: "hi"},
			{Path: "shallow.txt", Content: "ok"},
		},
		InitialDirs:  []string{"data", "data/raw"},
		ConfigMapRef: &corev1.LocalObjectReference{Name: "user-ws"},
		Bootstrap:    WorkspaceBootstrap{Enabled: Ptr(false)},
	}
	assert.Len(t, ws.InitialFiles, 2)
	assert.Equal(t, "notes/finance/2026.md", ws.InitialFiles[0].Path)
	assert.NotNil(t, ws.Bootstrap.Enabled)
	assert.False(t, *ws.Bootstrap.Enabled)
}
