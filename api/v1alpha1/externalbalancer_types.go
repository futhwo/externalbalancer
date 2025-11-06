package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Strategy determines how resources are rendered.
type Strategy string

const (
	StrategyWeightedPerService Strategy = "WeightedPerService"
	StrategySingleService      Strategy = "SingleService"
)

type ServiceTemplate struct {
	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

type Backend struct {
	Name             string `json:"name"`
	Address          string `json:"address"` // IP or DNS
	Port             int32  `json:"port"`
	Weight           *int32 `json:"weight,omitempty"`
	H2C              bool   `json:"h2c,omitempty"`
	StickyCookieName string `json:"stickyCookieName,omitempty"`
}

// MiddlewareRef references an existing Traefik Middleware by name/namespace.
type MiddlewareRef struct {
	Name      string `json:"name"`
	// Optional; defaults to the IngressRoute namespace when empty.
	Namespace string `json:"namespace,omitempty"`
}

// TLSConfig allows either a direct Secret or a cert-manager Certificate.
type TLSConfig struct {
	// SecretName: name of an existing Secret with TLS keypair.
	SecretName string `json:"secretName,omitempty"`
	// CertManager: when set, the operator will create/maintain a cert-manager Certificate.
	CertManager *CertManagerTLS `json:"certManager,omitempty"`
}

type CertManagerTLS struct {
	// SecretName to be created by cert-manager. If empty, defaults to "<metadata.name>-tls".
	SecretName string `json:"secretName,omitempty"`
	// IssuerRef selects an Issuer or ClusterIssuer.
	IssuerRef CertManagerIssuerRef `json:"issuerRef"`
	// Optional fields mirroring cert-manager.
	Duration    *metav1.Duration `json:"duration,omitempty"`
	RenewBefore *metav1.Duration `json:"renewBefore,omitempty"`
	Usages      []string         `json:"usages,omitempty"`
}

type CertManagerIssuerRef struct {
	Name  string `json:"name"`
	// Kind must be "Issuer" or "ClusterIssuer"; defaults to ClusterIssuer when empty.
	Kind  string `json:"kind,omitempty"`
	// Group defaults to "cert-manager.io" when empty.
	Group string `json:"group,omitempty"`
}

type ExternalBalancerSpec struct {
	// +kubebuilder:validation:Enum=WeightedPerService;SingleService
	// +kubebuilder:default=WeightedPerService
	Strategy Strategy `json:"strategy,omitempty"`

	// Labels applied only to the IngressRoute object
	IngressRouteLabels map[string]string `json:"ingressRouteLabels,omitempty"`

	Host        string   `json:"host"`
	EntryPoints []string `json:"entryPoints"`

	// TLS configuration. Exactly one of:
	// - tls.secretName
	// - tls.certManager
	TLS *TLSConfig `json:"tls"`

	// WRR top-level sticky cookie (WeightedPerService only)
	StickyCookieName string `json:"stickyCookieName,omitempty"`

	UseEndpointSlice bool            `json:"useEndpointSlice,omitempty"`
	ServiceTemplate  ServiceTemplate `json:"serviceTemplate,omitempty"`

	// +kubebuilder:validation:MinItems=1
	Backends []Backend `json:"backends"`

	// Middlewares to apply to the IngressRoute's route (existing Traefik Middleware references).
	Middlewares []MiddlewareRef `json:"middlewares,omitempty"`
}

type ExternalBalancerStatus struct {
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Ready              bool               `json:"ready,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
	ServicesCreated    int32              `json:"servicesCreated,omitempty"`
	EndpointsCreated   int32              `json:"endpointsCreated,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type ExternalBalancer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExternalBalancerSpec   `json:"spec,omitempty"`
	Status ExternalBalancerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type ExternalBalancerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ExternalBalancer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ExternalBalancer{}, &ExternalBalancerList{})
}

// ---- DeepCopy implementations (manual) ----

func (in *ServiceTemplate) DeepCopyInto(out *ServiceTemplate) {
	*out = *in
	if in.Annotations != nil {
		out.Annotations = make(map[string]string, len(in.Annotations))
		for k, v := range in.Annotations {
			out.Annotations[k] = v
		}
	}
	if in.Labels != nil {
		out.Labels = make(map[string]string, len(in.Labels))
		for k, v := range in.Labels {
			out.Labels[k] = v
		}
	}
}
func (in *ServiceTemplate) DeepCopy() *ServiceTemplate {
	if in == nil {
		return nil
	}
	out := new(ServiceTemplate)
	in.DeepCopyInto(out)
	return out
}

func (in *Backend) DeepCopyInto(out *Backend) {
	*out = *in
	if in.Weight != nil {
		w := *in.Weight
		out.Weight = &w
	}
}
func (in *Backend) DeepCopy() *Backend {
	if in == nil {
		return nil
	}
	out := new(Backend)
	in.DeepCopyInto(out)
	return out
}

func (in *ExternalBalancerSpec) DeepCopyInto(out *ExternalBalancerSpec) {
	*out = *in
	if in.IngressRouteLabels != nil {
		out.IngressRouteLabels = make(map[string]string, len(in.IngressRouteLabels))
		for k, v := range in.IngressRouteLabels {
			out.IngressRouteLabels[k] = v
		}
	}
	out.ServiceTemplate = ServiceTemplate{}
	in.ServiceTemplate.DeepCopyInto(&out.ServiceTemplate)
	if in.Backends != nil {
		out.Backends = make([]Backend, len(in.Backends))
		for i := range in.Backends {
			in.Backends[i].DeepCopyInto(&out.Backends[i])
		}
	}
	if in.TLS != nil {
		out.TLS = &TLSConfig{}
		in.TLS.DeepCopyInto(out.TLS)
	}
	if in.Middlewares != nil {
		out.Middlewares = make([]MiddlewareRef, len(in.Middlewares))
		copy(out.Middlewares, in.Middlewares)
	}
}
func (in *ExternalBalancerSpec) DeepCopy() *ExternalBalancerSpec {
	if in == nil {
		return nil
	}
	out := new(ExternalBalancerSpec)
	in.DeepCopyInto(out)
	return out
}

func (in *ExternalBalancerStatus) DeepCopyInto(out *ExternalBalancerStatus) {
	*out = *in
	if in.Conditions != nil {
		out.Conditions = make([]metav1.Condition, len(in.Conditions))
		for i := range in.Conditions {
			c := in.Conditions[i].DeepCopy()
			out.Conditions[i] = *c
		}
	}
}
func (in *ExternalBalancerStatus) DeepCopy() *ExternalBalancerStatus {
	if in == nil {
		return nil
	}
	out := new(ExternalBalancerStatus)
	in.DeepCopyInto(out)
	return out
}

func (in *ExternalBalancer) DeepCopyInto(out *ExternalBalancer) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}
func (in *ExternalBalancer) DeepCopy() *ExternalBalancer {
	if in == nil {
		return nil
	}
	out := new(ExternalBalancer)
	in.DeepCopyInto(out)
	return out
}
func (in *ExternalBalancer) DeepCopyObject() runtime.Object { return in.DeepCopy() }

func (in *ExternalBalancerList) DeepCopyInto(out *ExternalBalancerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]ExternalBalancer, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}
func (in *ExternalBalancerList) DeepCopy() *ExternalBalancerList {
	if in == nil {
		return nil
	}
	out := new(ExternalBalancerList)
	in.DeepCopyInto(out)
	return out
}
func (in *ExternalBalancerList) DeepCopyObject() runtime.Object { return in.DeepCopy() }

// DeepCopy helpers for TLS types (value-like)
func (in *MiddlewareRef) DeepCopyInto(out *MiddlewareRef) { *out = *in }
func (in *MiddlewareRef) DeepCopy() *MiddlewareRef {
	if in == nil { return nil }
	out := new(MiddlewareRef)
	*out = *in
	return out
}
func (in *TLSConfig) DeepCopyInto(out *TLSConfig) {
	*out = *in
	if in.CertManager != nil {
		out.CertManager = &CertManagerTLS{}
		in.CertManager.DeepCopyInto(out.CertManager)
	}
}
func (in *TLSConfig) DeepCopy() *TLSConfig {
	if in == nil { return nil }
	out := new(TLSConfig); in.DeepCopyInto(out); return out
}
func (in *CertManagerTLS) DeepCopyInto(out *CertManagerTLS) {
	*out = *in
	if in.Duration != nil { d := *in.Duration; out.Duration = &d }
	if in.RenewBefore != nil { r := *in.RenewBefore; out.RenewBefore = &r }
	if in.Usages != nil { out.Usages = append([]string{}, in.Usages...) }
}
func (in *CertManagerTLS) DeepCopy() *CertManagerTLS {
	if in == nil { return nil }
	out := new(CertManagerTLS); in.DeepCopyInto(out); return out
}
func (in *CertManagerIssuerRef) DeepCopyInto(out *CertManagerIssuerRef) { *out = *in }
func (in *CertManagerIssuerRef) DeepCopy() *CertManagerIssuerRef { if in == nil { return nil }; out := new(CertManagerIssuerRef); *out = *in; return out }
