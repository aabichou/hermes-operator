package resources

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	hermesv1 "github.com/stubbi/hermes-operator/api/v1"
)

// ServiceName returns the deterministic Service name for a HermesInstance.
func ServiceName(inst *hermesv1.HermesInstance) string { return inst.Name }

// BuildService returns the desired headless Service. Headless = stable DNS
// for the StatefulSet pod. Plan 2 adds optional ClusterIP / LoadBalancer modes.
func BuildService(inst *hermesv1.HermesInstance) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceName(inst),
			Namespace: inst.Namespace,
			Labels:    LabelsForInstance(inst),
		},
		Spec: corev1.ServiceSpec{
			ClusterIP:       corev1.ClusterIPNone,
			SessionAffinity: corev1.ServiceAffinityNone, // explicit k8s default
			Selector: map[string]string{
				"app.kubernetes.io/name":     "hermes-agent",
				"app.kubernetes.io/instance": inst.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "gateway",
					Port:       8443,
					TargetPort: intstr.FromString("gateway"),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}
