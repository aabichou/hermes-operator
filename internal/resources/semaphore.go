package resources

import (
	corev1 "k8s.io/api/core/v1"

	hermesv1 "github.com/paperclipinc/hermes-operator/api/v1"
)

// SemaphoreEnabled reports whether the user opted into Semaphore integration.
func SemaphoreEnabled(inst *hermesv1.HermesInstance) bool {
	return BoolValue(inst.Spec.Semaphore.Enabled)
}

// BuildSemaphoreEnv returns env vars injected into the hermes container
// when Semaphore integration is enabled. It also returns the semaphore-ui
// skill entry for auto-installation.
func BuildSemaphoreEnv(inst *hermesv1.HermesInstance) []corev1.EnvVar {
	if !SemaphoreEnabled(inst) {
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
	if !SemaphoreEnabled(inst) {
		return nil
	}
	return &hermesv1.InstanceSkill{
		Source: "semaphore-ui",
	}
}

// SemaphoreCleanupOnDelete reports whether the operator should clean up
// the Semaphore project and user when the HermesInstance is deleted.
func SemaphoreCleanupOnDelete(inst *hermesv1.HermesInstance) bool {
	return SemaphoreEnabled(inst) && BoolValue(inst.Spec.Semaphore.CleanupOnDelete)
}

// SemaphoreSoulBlock returns the markdown block injected into SOUL.md
// when Semaphore integration is enabled. It includes the API quickstart
// and skill loading instructions.
const SemaphoreSoulBlock = `

# Semaphore UI — infrastructure automation

You have access to a **Semaphore UI** instance for running Ansible
playbooks, Terraform/OpenTofu plans, and general scripts with full
run logs and audit trails.

- **URL**: ` + "`$SEMAPHORE_URL`" + ` (injected automatically)
- **Auth**: ` + "`$SEMAPHORE_TOKEN`" + ` (per-agent, scoped to this project)
- **Skill**: Load the ` + "`semaphore-ui`" + ` skill (` + "`skill_view(name='semaphore-ui')`" + `)
  for the full API reference, then use Semaphore for any multi-step
  infrastructure operation that needs logging, rollback, or approval gates.

When given an infra task that spans multiple hosts or needs an audit trail
(Terraform, Ansible, DB migration, cluster upgrade), **default to Semaphore**.
The ` + "`semaphore-ui`" + ` skill has templates for common tasks.

**Quickstart** (fallback if skill not loaded):

` + "```" + `python
import http.client, json, os

url = os.environ["SEMAPHORE_URL"]
token = os.environ["SEMAPHORE_TOKEN"]
host = url.split("://")[1].split(":")[0]
port = int(url.split(":")[-1])

def sema(method, path, body=None):
    c = http.client.HTTPConnection(host, port)
    h = {"Authorization": f"Bearer {token}"}
    if body:
        h["Content-Type"] = "application/json"
        body = json.dumps(body)
    c.request(method, path, body, h)
    return json.loads(c.getresponse().read())

# your projects (token-scoped)
print(sema("GET", "/api/projects"))
` + "```" + `
`

// (re-base verified against upstream v0.1.19 — see FORK_NOTES)
