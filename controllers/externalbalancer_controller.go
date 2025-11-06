package controllers

import (
	ccontext "context"
	"fmt"
	"sort"

	netv1alpha1 "github.com/futhwo/externalbalancer/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controllerutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const finalizerName = "externalbalancer.net.futhwo.io/finalizer"

type ExternalBalancerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=net.futhwo.io,resources=externalbalancers,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=net.futhwo.io,resources=externalbalancers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=services;endpoints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="discovery.k8s.io",resources=endpointslices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="traefik.io",resources=ingressroutes;traefikservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="cert-manager.io",resources=certificates,verbs=get;list;watch;create;update;patch;delete

func (r *ExternalBalancerReconciler) Reconcile(ctx ccontext.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var eb netv1alpha1.ExternalBalancer
	if err := r.Get(ctx, req.NamespacedName, &eb); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle deletion
	if eb.DeletionTimestamp != nil {
		controllerutil.RemoveFinalizer(&eb, finalizerName)
		_ = r.Update(ctx, &eb)
		return ctrl.Result{}, nil
	}

	// Ensure finalizer
	if !controllerutil.ContainsFinalizer(&eb, finalizerName) {
		controllerutil.AddFinalizer(&eb, finalizerName)
		if err := r.Update(ctx, &eb); err != nil {
			return ctrl.Result{}, err
		}
	}

	// helper: build middlewares array for Traefik unstructured specs
	buildMiddlewaresArray := func(eb *netv1alpha1.ExternalBalancer) []any {
		if len(eb.Spec.Middlewares) == 0 {
			return nil
		}
		out := make([]any, 0, len(eb.Spec.Middlewares))
		for _, m := range eb.Spec.Middlewares {
			item := map[string]any{"name": m.Name}
			if m.Namespace != "" {
				item["namespace"] = m.Namespace
			}
			out = append(out, item)
		}
		return out
	}

	// helper: ensure TLS and return the secretName to reference from IngressRoute
	ensureTLS := func(ctx ccontext.Context, eb *netv1alpha1.ExternalBalancer) (string, error) {
		if eb.Spec.TLS == nil {
			return "", fmt.Errorf("spec.tls is required")
		}
		// Case 1: direct Secret
		if eb.Spec.TLS.SecretName != "" {
			return eb.Spec.TLS.SecretName, nil
		}
		// Case 2: cert-manager Certificate
		cm := eb.Spec.TLS.CertManager
		if cm == nil {
			return "", fmt.Errorf("exactly one of tls.secretName or tls.certManager must be set")
		}
		secName := cm.SecretName
		if secName == "" {
			secName = eb.Name + "-tls"
		}
		issuerKind := cm.IssuerRef.Kind
		if issuerKind == "" {
			issuerKind = "ClusterIssuer"
		}
		issuerGroup := cm.IssuerRef.Group
		if issuerGroup == "" {
			issuerGroup = "cert-manager.io"
		}
		cert := &unstructured.Unstructured{}
		cert.SetGroupVersionKind(schema.GroupVersionKind{Group: "cert-manager.io", Version: "v1", Kind: "Certificate"})
		cert.SetName(eb.Name)
		cert.SetNamespace(eb.Namespace)
		_, err := createOrUpdateUnstructured(ctx, r.Client, cert, func(o *unstructured.Unstructured) error {
			spec := map[string]any{
				"secretName": secName,
				"dnsNames":   []any{eb.Spec.Host},
				"issuerRef":  map[string]any{"name": cm.IssuerRef.Name, "kind": issuerKind, "group": issuerGroup},
			}
			if cm.Duration != nil { spec["duration"] = cm.Duration.Duration.String() }
			if cm.RenewBefore != nil { spec["renewBefore"] = cm.RenewBefore.Duration.String() }
			if len(cm.Usages) > 0 { arr := make([]any, 0, len(cm.Usages)); for _, u := range cm.Usages { arr = append(arr, u) }; spec["usages"] = arr }
			o.Object["spec"] = spec
			return controllerutil.SetControllerReference(&eb, o, r.Scheme)
		})
		return secName, err
	}

	strategy := eb.Spec.Strategy
	if strategy == "" {
		strategy = netv1alpha1.StrategyWeightedPerService
	}

	var createdSvcs, createdEps int32

	switch strategy {
	case netv1alpha1.StrategyWeightedPerService:
		for _, b := range eb.Spec.Backends {
			// Service (no selector)
			svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: b.Name, Namespace: eb.Namespace}}
			_, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
				mergeStringMap(&svc.Labels, eb.Spec.ServiceTemplate.Labels)
				mergeStringMap(&svc.Annotations, eb.Spec.ServiceTemplate.Annotations)
				svc.Spec.Selector = nil
				svc.Spec.Ports = []corev1.ServicePort{{
					Port:       b.Port,
					TargetPort: intstr.FromInt(int(b.Port)),
					Protocol:   corev1.ProtocolTCP,
					AppProtocol: func() *string {
						if b.H2C {
							v := "kubernetes.io/h2c"
							return &v
						}
						return nil
					}(),
				}}
				return controllerutil.SetControllerReference(&eb, svc, r.Scheme)
			})
			if err != nil { return ctrl.Result{}, err }
			createdSvcs++

			// Endpoints or EndpointSlice
			if eb.Spec.UseEndpointSlice {
				es := &discoveryv1.EndpointSlice{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-1", b.Name),
						Namespace: eb.Namespace,
						Labels: map[string]string{
							discoveryv1.LabelServiceName: b.Name,
						},
					},
				}
				_, err = controllerutil.CreateOrUpdate(ctx, r.Client, es, func() error {
					es.AddressType = discoveryv1.AddressTypeIPv4
					portNum := int32(b.Port)
					var ap *string
					if b.H2C {
						v := "kubernetes.io/h2c"
						ap = &v
					}
					es.Ports = []discoveryv1.EndpointPort{{Port: &portNum, AppProtocol: ap}}
					ready := true
					es.Endpoints = []discoveryv1.Endpoint{{
						Addresses:  []string{b.Address},
						Conditions: discoveryv1.EndpointConditions{Ready: &ready},
					}}
					return controllerutil.SetControllerReference(&eb, es, r.Scheme)
				})
				if err != nil { return ctrl.Result{}, err }
			} else {
				ep := &corev1.Endpoints{ObjectMeta: metav1.ObjectMeta{Name: b.Name, Namespace: eb.Namespace}}
				_, err = controllerutil.CreateOrUpdate(ctx, r.Client, ep, func() error {
					var ap *string
					if b.H2C {
						v := "kubernetes.io/h2c"
						ap = &v
					}
					ep.Subsets = []corev1.EndpointSubset{{
						Addresses: []corev1.EndpointAddress{{IP: b.Address}},
						Ports: []corev1.EndpointPort{{
							Port:        int32(b.Port),
							Protocol:    corev1.ProtocolTCP,
							AppProtocol: ap,
						}},
					}}
					return controllerutil.SetControllerReference(&eb, ep, r.Scheme)
				})
				if err != nil { return ctrl.Result{}, err }
			}
			createdEps++
		}

		// TraefikService (WRR) via unstructured
		wrrName := fmt.Sprintf("%s-wrr", eb.Name)
		ts := &unstructured.Unstructured{}
		ts.SetGroupVersionKind(schema.GroupVersionKind{Group: "traefik.io", Version: "v1alpha1", Kind: "TraefikService"})
		ts.SetName(wrrName)
		ts.SetNamespace(eb.Namespace)
		_, err := createOrUpdateUnstructured(ctx, r.Client, ts, func(obj *unstructured.Unstructured) error {
			var services []map[string]any
			for _, b := range eb.Spec.Backends {
				w := 1
				if b.Weight != nil {
					w = int(*b.Weight)
				}
				item := map[string]any{
					"name":   b.Name,
					"kind":   "Service",
					"port":   b.Port,
					"weight": w,
				}
				if b.StickyCookieName != "" {
					item["sticky"] = map[string]any{"cookie": map[string]any{"name": b.StickyCookieName}}
				}
				services = append(services, item)
			}
			sort.SliceStable(services, func(i, j int) bool { return services[i]["name"].(string) < services[j]["name"].(string) })
			spec := map[string]any{"weighted": map[string]any{"services": services}}
			if eb.Spec.StickyCookieName != "" {
				spec["weighted"].(map[string]any)["sticky"] = map[string]any{"cookie": map[string]any{"name": eb.Spec.StickyCookieName}}
			}
			obj.Object["spec"] = spec
			return controllerutil.SetControllerReference(&eb, obj, r.Scheme)
		})
		if err != nil { return ctrl.Result{}, err }

		// IngressRoute -> TraefikService
		tlsSecretName, err := ensureTLS(ctx, &eb)
		if err != nil { return ctrl.Result{}, err }
		mw := buildMiddlewaresArray(&eb)
		ir := &unstructured.Unstructured{}
		ir.SetGroupVersionKind(schema.GroupVersionKind{Group: "traefik.io", Version: "v1alpha1", Kind: "IngressRoute"})
		ir.SetName(eb.Name)
		ir.SetNamespace(eb.Namespace)
		_, err = createOrUpdateUnstructured(ctx, r.Client, ir, func(obj *unstructured.Unstructured) error {
			mergeMetadataLabels(obj, eb.Spec.IngressRouteLabels)
			route := map[string]any{
				"kind":     "Rule",
				"match":    fmt.Sprintf("Host(`%s`)", eb.Spec.Host),
				"priority": 1,
				"services": []any{
					map[string]any{
						"kind": "TraefikService",
						"name": wrrName,
					},
				},
			}
			if len(mw) > 0 {
				route["middlewares"] = mw
			}
			obj.Object["spec"] = map[string]any{
				"entryPoints": eb.Spec.EntryPoints,
				"routes":      []any{route},
				"tls":         map[string]any{"secretName": tlsSecretName},
			}
			return controllerutil.SetControllerReference(&eb, obj, r.Scheme)
		})
		if err != nil { return ctrl.Result{}, err }

	case netv1alpha1.StrategySingleService:
		// Validate same port across backends
		commonPort := eb.Spec.Backends[0].Port
		for _, b := range eb.Spec.Backends {
			if b.Port != commonPort {
				return ctrl.Result{}, fmt.Errorf("all backends must share the same port in SingleService strategy")
			}
		}

		svcName := eb.Name + "-svc"
		// Service
		svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: svcName, Namespace: eb.Namespace}}
		_, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
			mergeStringMap(&svc.Labels, eb.Spec.ServiceTemplate.Labels)
			mergeStringMap(&svc.Annotations, eb.Spec.ServiceTemplate.Annotations)
			svc.Spec.Selector = nil
			var ap *string
			for _, b := range eb.Spec.Backends {
				if b.H2C {
					v := "kubernetes.io/h2c"
					ap = &v
					break
				}
			}
			svc.Spec.Ports = []corev1.ServicePort{{
				Port:        commonPort,
				TargetPort:  intstr.FromInt(int(commonPort)),
				Protocol:    corev1.ProtocolTCP,
				AppProtocol: ap,
			}}
			return controllerutil.SetControllerReference(&eb, svc, r.Scheme)
		})
		if err != nil { return ctrl.Result{}, err }
		createdSvcs++

		// Aggregate endpoints
		if eb.Spec.UseEndpointSlice {
			es := &discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:      svcName + "-1",
					Namespace: eb.Namespace,
					Labels:    map[string]string{discoveryv1.LabelServiceName: svcName},
				},
			}
			_, err = controllerutil.CreateOrUpdate(ctx, r.Client, es, func() error {
				es.AddressType = discoveryv1.AddressTypeIPv4
				portNum := int32(commonPort)
				es.Ports = []discoveryv1.EndpointPort{{Port: &portNum}}
				ready := true
				var eps []discoveryv1.Endpoint
				for _, b := range eb.Spec.Backends {
					eps = append(eps, discoveryv1.Endpoint{
						Addresses:  []string{b.Address},
						Conditions: discoveryv1.EndpointConditions{Ready: &ready},
					})
				}
				es.Endpoints = eps
				return controllerutil.SetControllerReference(&eb, es, r.Scheme)
			})
			if err != nil { return ctrl.Result{}, err }
		} else {
			ep := &corev1.Endpoints{ObjectMeta: metav1.ObjectMeta{Name: svcName, Namespace: eb.Namespace}}
			_, err = controllerutil.CreateOrUpdate(ctx, r.Client, ep, func() error {
				sub := corev1.EndpointSubset{
					Ports: []corev1.EndpointPort{{Port: int32(commonPort), Protocol: corev1.ProtocolTCP}},
				}
				for _, b := range eb.Spec.Backends {
					sub.Addresses = append(sub.Addresses, corev1.EndpointAddress{IP: b.Address})
				}
				ep.Subsets = []corev1.EndpointSubset{sub}
				return controllerutil.SetControllerReference(&eb, ep, r.Scheme)
			})
			if err != nil { return ctrl.Result{}, err }
		}
		createdEps++

		// IngressRoute -> Service
		tlsSecretName, err := ensureTLS(ctx, &eb)
		if err != nil { return ctrl.Result{}, err }
		mw := buildMiddlewaresArray(&eb)
		ir := &unstructured.Unstructured{}
		ir.SetGroupVersionKind(schema.GroupVersionKind{Group: "traefik.io", Version: "v1alpha1", Kind: "IngressRoute"})
		ir.SetName(eb.Name)
		ir.SetNamespace(eb.Namespace)
		_, err = createOrUpdateUnstructured(ctx, r.Client, ir, func(obj *unstructured.Unstructured) error {
			mergeMetadataLabels(obj, eb.Spec.IngressRouteLabels)
			route := map[string]any{
				"kind":     "Rule",
				"match":    fmt.Sprintf("Host(`%s`)", eb.Spec.Host),
				"priority": 1,
				"services": []any{map[string]any{"kind": "Service", "name": svcName, "port": commonPort}},
			}
			if len(mw) > 0 {
				route["middlewares"] = mw
			}
			obj.Object["spec"] = map[string]any{
				"entryPoints": eb.Spec.EntryPoints,
				"routes":      []any{route},
				"tls":         map[string]any{"secretName": tlsSecretName},
			}
			return controllerutil.SetControllerReference(&eb, obj, r.Scheme)
		})
		if err != nil { return ctrl.Result{}, err }
	default:
		return ctrl.Result{}, fmt.Errorf("unknown strategy %q", strategy)
	}

	// status
	eb.Status.ObservedGeneration = eb.Generation
	eb.Status.Ready = true
	eb.Status.ServicesCreated = createdSvcs
	eb.Status.EndpointsCreated = createdEps
	if err := r.Status().Update(ctx, &eb); err != nil {
		logger.Error(err, "status update failed")
	}

	return ctrl.Result{}, nil
}

func (r *ExternalBalancerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netv1alpha1.ExternalBalancer{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Endpoints{}).
		Owns(&discoveryv1.EndpointSlice{}).
		Complete(r)
}

func mergeStringMap(dst *map[string]string, src map[string]string) {
	if src == nil {
		return
	}
	if *dst == nil {
		*dst = map[string]string{}
	}
	for k, v := range src {
		(*dst)[k] = v
	}
}

// createOrUpdateUnstructured provides a minimal "create or update" for unstructured objects.
func createOrUpdateUnstructured(ctx ccontext.Context, c client.Client, obj *unstructured.Unstructured, mutate func(*unstructured.Unstructured) error) (controllerutil.OperationResult, error) {
	key := types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(obj.GroupVersionKind())

	err := c.Get(ctx, key, existing)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := mutate(obj); err != nil {
				return controllerutil.OperationResultNone, err
			}
			if err := c.Create(ctx, obj); err != nil {
				return controllerutil.OperationResultNone, err
			}
			return controllerutil.OperationResultCreated, nil
		}
		return controllerutil.OperationResultNone, err
	}

	// copy metadata into obj to keep resourceVersion etc
	obj.SetResourceVersion(existing.GetResourceVersion())
	obj.SetUID(existing.GetUID())
	obj.SetCreationTimestamp(existing.GetCreationTimestamp())
	obj.SetManagedFields(nil)

	if err := mutate(obj); err != nil {
		return controllerutil.OperationResultNone, err
	}
	if err := c.Update(ctx, obj); err != nil {
		return controllerutil.OperationResultNone, err
	}
	return controllerutil.OperationResultUpdated, nil
}

// mergeMetadataLabels merges labels into unstructured object's metadata.labels
func mergeMetadataLabels(obj *unstructured.Unstructured, labels map[string]string) {
	if len(labels) == 0 {
		return
	}
	m := obj.Object
	metaI, ok := m["metadata"]
	if !ok || metaI == nil {
		metaI = map[string]any{}
		m["metadata"] = metaI
	}
	meta := metaI.(map[string]any)
	lblI, ok := meta["labels"]
	if !ok || lblI == nil {
		lblI = map[string]any{}
		meta["labels"] = lblI
	}
	lbl := lblI.(map[string]any)
	for k, v := range labels {
		lbl[k] = v
	}
}
