package resources

import (
	corev1 "k8s.io/api/core/v1"

	hermesv1 "github.com/paperclipinc/hermes-operator/api/v1"
)

// semaphoreEnabled reports whether the user opted into Semaphore integration.
func semaphoreEnabled(inst *hermesv1.HermesInstance) bool {
	return BoolValue(inst.Spec.Semaphore.Enabled)
}

// BuildSemaphoreEnv returns env vars injected into the hermes container
// when Semaphore integration is enabled. It also returns the semaphore-ui
// skill entry for auto-installation.
func BuildSemaphoreEnv(inst *hermesv1.HermesInstance) []corev1.EnvVar {
	if !semaphoreEnabled(inst) {
		return nil
	}
	s := inst.Spec.Semaphore

	out := []corev1.EnvVar{
		{Name: "SEMAPHORE_URL", Value: s.URL},
	}
	if s.TokenSecretRef != nil {
		out = append(out, corev1.EnvVar{
			Name: "SEMAPHORE_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: s.TokenSecretRef.LocalObjectReference,
					Key:                  s.TokenSecretRef.Key,
				},
			},
		})
	}
	return out
}

// BuildSemaphoreSkill returns the semaphore-ui skill entry for the
// HermesInstance Skills list when Semaphore integration is enabled.
func BuildSemaphoreSkill(inst *hermesv1.HermesInstance) *hermesv1.InstanceSkill {
	if !semaphoreEnabled(inst) {
		return nil
	}
	return &hermesv1.InstanceSkill{
		Source: "semaphore-ui",
	}
}
