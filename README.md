# extlb-operator

An example Kubernetes operator that automates Traefik-based external load-balancing
using a single Custom Resource: **ExternalBalancer**.

It supports two strategies:

- **WeightedPerService** (legacy/sticky): one Service per backend + Endpoints/EndpointSlices,
  a TraefikService (WRR) with optional per-service and group-level stickiness, and an IngressRoute
  pointing to the TraefikService.

- **SingleService** (lean/modern): one Service + one Endpoints/EndpointSlice aggregating all backends,
  and an IngressRoute pointing directly to the Service. No TraefikService and no stickiness.

It also supports **IngressRoute-only labels** (for selecting a specific Traefik instance that
watches only IngressRoutes with certain labels).

> NOTE: For simplicity and portability, this project manages Traefik CRDs (**IngressRoute** and
> **TraefikService**) using `unstructured.Unstructured`, so you don't need to pull in Traefik's Go types.

## Quick start (developer workflow)

- Make sure you have Go 1.22+ and Kubernetes 1.26+.
- Create a kind/minikube cluster (optional).
- Install Traefik CRDs (via the Traefik Helm chart or manifests).
- Apply the CRD provided here:
  ```bash
  kubectl apply -f config/crd/bases/net.futhwo.io_externalbalancers.yaml
  ```
- Build and run the operator locally against your cluster:
  ```bash
  go run ./main.go
  ```
  Or build a container, push, and run it in-cluster with an RBAC and Deployment (left as an exercise).

- Apply sample CRs (choose one):
  ```bash
  kubectl apply -f config/samples/net_v1alpha1_externalbalancer_weighted.yaml
  kubectl apply -f config/samples/net_v1alpha1_externalbalancer_single.yaml
  ```

## CRD

See `config/crd/bases/net.futhwo.io_externalbalancers.yaml`.

## Notes

- When `strategy: SingleService`, the operator validates that all backends share the same port.
- `ingressRouteLabels` are applied **only** to the IngressRoute object.
- Use `useEndpointSlice: true` when possible (scales better and supports DNS addresses cleanly).
