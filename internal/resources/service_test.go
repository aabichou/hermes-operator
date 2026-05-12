package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	hermesv1 "github.com/stubbi/hermes-operator/api/v1"
)

func TestBuildService_DefaultsAndSelector(t *testing.T) {
	inst := &hermesv1.HermesInstance{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "agents"},
	}
	svc := BuildService(inst)
	assert.Equal(t, "demo", svc.Name)
	assert.Equal(t, corev1.ClusterIPNone, svc.Spec.ClusterIP, "headless Service for StatefulSet")
	assert.Equal(t, corev1.ServiceAffinityNone, svc.Spec.SessionAffinity, "explicit k8s default")
	assert.Equal(t, "demo", svc.Spec.Selector["app.kubernetes.io/instance"])
	// Must declare at least the gateway port so the Service is valid.
	assert.NotEmpty(t, svc.Spec.Ports)
}
