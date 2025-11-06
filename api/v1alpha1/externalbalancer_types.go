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
