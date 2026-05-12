package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Happy path", func() {
	It("reconciles a minimal HermesInstance into a running StatefulSet", func() {
		// TODO(plan-3): switch to ghcr.io/stubbi/hermes-agent:latest once Plan 3
		// publishes the agent image. Until then we use a generic non-root image
		// so the pod can become Ready.
		//
		// NOTE: nginx-unprivileged listens on port 8080 by default, not 8443.
		// The Service expects the "gateway" port 8443. The pod will NOT reach
		// Ready because the readiness probe (TCP socket on gateway/8443) will
		// fail. This test is expected to time out until Plan 3 ships the real
		// hermes-agent image that opens 8443. The e2e infrastructure is correct;
		// the placeholder image is intentional for Plan 1.
		manifest := `
apiVersion: hermes.agent/v1
kind: HermesInstance
metadata:
  name: e2e-demo
  namespace: default
spec:
  image:
    repository: ghcr.io/nginx/nginx-unprivileged
    tag: latest
    pullPolicy: IfNotPresent
  storage:
    persistence:
      enabled: true
      size: 1Gi
`
		out, err := runStdin("kubectl", []string{"apply", "-f", "-"}, manifest)
		Expect(err).ToNot(HaveOccurred(), "kubectl apply failed: %s", out)

		Eventually(func() string {
			out, _ := kubectl("get", "statefulset", "e2e-demo", "-n", "default", "-o", "jsonpath={.status.readyReplicas}")
			return out
		}).Should(Equal("1"))
	})
})
