package v1alpha1

import (
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
	// name of the per-backend Service in WeightedPerService, or logical id in SingleService
	Name string `json:"name"`
	// Address is IP or DNS name
	Address string `json:"address"`
	Port    int32  `json:"port"`
	// Weight (WeightedPerService only)
	Weight *int32 `json:"weight,omitempty"`
	// H2C indicates cleartext HTTP/2
	H2C bool `json:"h2c,omitempty"`
	// StickyCookieName (WeightedPerService only)
	StickyCookieName string `json:"stickyCookieName,omitempty"`
}

type ExternalBalancerSpec struct {
	// +kubebuilder:validation:Enum=WeightedPerService;SingleService
	// +kubebuilder:default=WeightedPerService
	Strategy Strategy `json:"strategy,omitempty"`

	// Labels applied only to the IngressRoute object
	IngressRouteLabels map[string]string `json:"ingressRouteLabels,omitempty"`

	Host          string   `json:"host"`
	EntryPoints   []string `json:"entryPoints"`
	TlsSecretName string   `json:"tlsSecretName"`

	// WRR top-level sticky cookie (WeightedPerService only)
	StickyCookieName string `json:"stickyCookieName,omitempty"`

	UseEndpointSlice bool            `json:"useEndpointSlice,omitempty"`
	ServiceTemplate  ServiceTemplate `json:"serviceTemplate,omitempty"`

	// +kubebuilder:validation:MinItems=1
	Backends []Backend `json:"backends"`
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
