# AI Agent Instructions for extlb-operator

## Project Overview
This is a Kubernetes operator that manages Traefik-based external load balancing using a Custom Resource Definition (CRD) called `ExternalBalancer`. The operator creates and manages Kubernetes and Traefik resources to implement load balancing strategies.

## Key Architecture Points
1. **Custom Resource Definition (CRD)**:
   - Located in `config/crd/bases/net.futhwo.io_externalbalancers.yaml`
   - Defines the `ExternalBalancer` spec with two strategies: `WeightedPerService` and `SingleService`
   - API types defined in `api/v1alpha1/externalbalancer_types.go`

````instructions
# AI agent instructions — extlb-operator (concise reference)

Overview
- This project is a Kubernetes operator that provisions Traefik resources for external load balancing using a CRD named `ExternalBalancer`.

Key files & entry points
- `api/v1alpha1/externalbalancer_types.go` — CRD Go types and validation (Strategy, Backend, ServiceTemplate).
- `controllers/externalbalancer_controller.go` — reconciliation logic: creates Services, Endpoints/EndpointSlices, Traefik `IngressRoute` and `TraefikService` (unstructured).
- `config/crd/bases/net.futhwo.io_externalbalancers.yaml` — CRD manifest.
- `config/samples/` — sample CRs for `WeightedPerService` and `SingleService`.
- `Makefile` — convenient targets: `make run`, `make crd`, `make samples`.

Quick dev commands (copyable)
```
# apply CRD
make crd

# run operator locally
make run    # runs: GO111MODULE=on go run ./main.go

# apply sample CRs
make samples
```

Important project patterns (do not change lightly)
- Uses controller-runtime `controllerutil.CreateOrUpdate` for typed K8s resources (Services, Endpoints, EndpointSlices).
- Traefik CRDs are handled as `unstructured.Unstructured` via a local helper `createOrUpdateUnstructured(...)` in the controller — mutate the `obj.Object["spec"]` map inside that mutate function.
- Controller sets owner references for K8s resources via `controllerutil.SetControllerReference(&eb, obj, r.Scheme)` so garbage collection works. Traefik unstructured objects are not `Owns()`-registered (so tests/changes should respect that).
- Finalizer handling: `finalizerName = "externalbalancer.net.futhwo.io/finalizer"` is used; reconcile returns errors to requeue.

Traefik specifics
- Traefik resources created:
  - TraefikService WRR named `fmt.Sprintf("%s-wrr", eb.Name)` for WeightedPerService.
  - IngressRoute named `eb.Name` referencing either the WRR TraefikService or the Service (SingleService).
- When editing Traefik payloads, modify the controller's mutate closures (search for `TraefikService` / `IngressRoute` blocks) and `schemaGVK(...)` helper.

Behavioral rules enforced in code
- `SingleService` requires all backends to share the same port — the controller validates and returns an error otherwise.
- `UseEndpointSlice` toggles whether EndpointSlice or Endpoints are created.
- H2C handling: backend `H2C` boolean adds AppProtocol `kubernetes.io/h2c` to ports if true.

How to extend safely
- Add new fields to the API in `api/v1alpha1/*` and update the controller to populate resources; update `config/samples/`.
- For Traefik features, prefer changing the unstructured spec builder and keep labels/annotations merged via `mergeMetadataLabels` / `mergeStringMap` helpers.

Environment & requirements
- Go >= 1.22, Kubernetes >= 1.26. Traefik CRDs must be installed in the cluster where samples are applied.

Where to look for tests & checks
- There are no unit tests here; run `go build ./...` and `go vet ./...` locally to catch compile issues after edits.

If something is unclear or you need more detail (example CRs, sample payloads, or tests), say which area to expand.
````