package v1

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
