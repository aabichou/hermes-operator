package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	hermesv1 "github.com/stubbi/hermes-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildNetworkPolicy_DenyAllBase(t *testing.T) {
	t.Parallel()
	inst := &hermesv1.HermesInstance{ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "agents"}}
	np := BuildNetworkPolicy(inst)
	assert.Equal(t, "demo", np.Name)
	assert.Equal(t, "agents", np.Namespace)
	assert.Contains(t, np.Spec.PolicyTypes, networkingv1.PolicyTypeIngress)
	assert.Contains(t, np.Spec.PolicyTypes, networkingv1.PolicyTypeEgress)
	assert.Equal(t, "demo", np.Spec.PodSelector.MatchLabels["app.kubernetes.io/instance"])
	assert.Equal(t, "hermes-agent", np.Spec.PodSelector.MatchLabels["app.kubernetes.io/name"])
}

func TestBuildNetworkPolicy_SameNamespaceIngress(t *testing.T) {
	t.Parallel()
	inst := &hermesv1.HermesInstance{ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "agents"}}
	np := BuildNetworkPolicy(inst)
	require := false
	for _, rule := range np.Spec.Ingress {
		for _, from := range rule.From {
			if from.NamespaceSelector != nil &&
				from.NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"] == "agents" {
				require = true
			}
		}
	}
	assert.True(t, require, "expected ingress rule from same namespace")
}

func TestBuildNetworkPolicy_DefaultDNSEgress(t *testing.T) {
	t.Parallel()
	inst := &hermesv1.HermesInstance{ObjectMeta: metav1.ObjectMeta{Name: "demo"}}
	np := BuildNetworkPolicy(inst)
	foundUDP53, foundTCP443 := false, false
	for _, rule := range np.Spec.Egress {
		for _, p := range rule.Ports {
			if p.Protocol != nil && *p.Protocol == corev1.ProtocolUDP && p.Port != nil && p.Port.IntValue() == 53 {
				foundUDP53 = true
			}
			if p.Protocol != nil && *p.Protocol == corev1.ProtocolTCP && p.Port != nil && p.Port.IntValue() == 443 {
				foundTCP443 = true
			}
		}
	}
	assert.True(t, foundUDP53, "default-allow DNS UDP/53")
	assert.True(t, foundTCP443, "default-allow HTTPS TCP/443")
}

func TestBuildNetworkPolicy_AllowDNSDisabled(t *testing.T) {
	t.Parallel()
	inst := &hermesv1.HermesInstance{
		ObjectMeta: metav1.ObjectMeta{Name: "demo"},
		Spec: hermesv1.HermesInstanceSpec{
			Security: hermesv1.SecuritySpec{
				NetworkPolicy: hermesv1.NetworkPolicySpec{AllowDNS: Ptr(false)},
			},
		},
	}
	np := BuildNetworkPolicy(inst)
	for _, rule := range np.Spec.Egress {
		for _, p := range rule.Ports {
			if p.Port != nil && p.Port.IntValue() == 53 {
				t.Fatalf("expected no DNS rule when AllowDNS=false")
			}
		}
	}
}

func TestBuildNetworkPolicy_AllowedIngressNamespacesAndCIDRs(t *testing.T) {
	t.Parallel()
	inst := &hermesv1.HermesInstance{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "agents"},
		Spec: hermesv1.HermesInstanceSpec{
			Security: hermesv1.SecuritySpec{
				NetworkPolicy: hermesv1.NetworkPolicySpec{
					AllowedIngressNamespaces: []string{"prometheus"},
					AllowedIngressCIDRs:      []string{"10.0.0.0/8"},
				},
			},
		},
	}
	np := BuildNetworkPolicy(inst)
	var sawNS, sawCIDR bool
	for _, rule := range np.Spec.Ingress {
		for _, from := range rule.From {
			if from.NamespaceSelector != nil &&
				from.NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"] == "prometheus" {
				sawNS = true
			}
			if from.IPBlock != nil && from.IPBlock.CIDR == "10.0.0.0/8" {
				sawCIDR = true
			}
		}
	}
	assert.True(t, sawNS, "expected ingress rule for namespace prometheus")
	assert.True(t, sawCIDR, "expected ingress rule for CIDR 10.0.0.0/8")
}

func TestBuildNetworkPolicy_AdditionalEgress(t *testing.T) {
	t.Parallel()
	extra := networkingv1.NetworkPolicyEgressRule{
		To: []networkingv1.NetworkPolicyPeer{{IPBlock: &networkingv1.IPBlock{CIDR: "203.0.113.0/24"}}},
	}
	inst := &hermesv1.HermesInstance{
		ObjectMeta: metav1.ObjectMeta{Name: "demo"},
		Spec: hermesv1.HermesInstanceSpec{
			Security: hermesv1.SecuritySpec{
				NetworkPolicy: hermesv1.NetworkPolicySpec{AdditionalEgress: []networkingv1.NetworkPolicyEgressRule{extra}},
			},
		},
	}
	np := BuildNetworkPolicy(inst)
	var sawExtra bool
	for _, rule := range np.Spec.Egress {
		for _, peer := range rule.To {
			if peer.IPBlock != nil && peer.IPBlock.CIDR == "203.0.113.0/24" {
				sawExtra = true
			}
		}
	}
	assert.True(t, sawExtra)
}

func TestNetworkPolicyName(t *testing.T) {
	t.Parallel()
	inst := &hermesv1.HermesInstance{ObjectMeta: metav1.ObjectMeta{Name: "demo"}}
	assert.Equal(t, "demo", NetworkPolicyName(inst))
	_ = metav1.ObjectMeta{}
}
