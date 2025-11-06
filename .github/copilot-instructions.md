# AI Agent Instructions for extlb-operator

## Project Overview
This is a Kubernetes operator that manages Traefik-based external load balancing using a Custom Resource Definition (CRD) called `ExternalBalancer`. The operator creates and manages Kubernetes and Traefik resources to implement load balancing strategies.

## Key Architecture Points
1. **Custom Resource Definition (CRD)**:
   - Located in `config/crd/bases/net.futhwo.io_externalbalancers.yaml`
   - Defines the `ExternalBalancer` spec with two strategies: `WeightedPerService` and `SingleService`
   - API types defined in `api/v1alpha1/externalbalancer_types.go`

2. **Controller Logic**:
   - Main reconciliation loop in `controllers/externalbalancer_controller.go`
   - Manages lifecycle of dependent resources:
     - Kubernetes Services and Endpoints/EndpointSlices
     - Traefik IngressRoutes and TraefikServices (using unstructured.Unstructured)

## Key Development Patterns
1. **Resource Management**:
   - Uses controller-runtime's `CreateOrUpdate` pattern for Kubernetes resources
   - Custom `createOrUpdateUnstructured` for Traefik CRDs to avoid dependency on Traefik types
   - All resources are owned by the ExternalBalancer CR for garbage collection

2. **Strategy Implementation**:
   - `WeightedPerService`: Creates per-backend Services + TraefikService with WRR
   - `SingleService`: Creates single Service aggregating all backends
   - See sample CRs in `config/samples/` for examples of each strategy

3. **Code Organization**:
   - API types and validation in `api/v1alpha1/`
   - Controller logic in `controllers/`
   - CRD manifests in `config/crd/`
   - RBAC rules in `config/rbac/`

## Development Workflow
1. **Local Development**:
   ```bash
   # Apply CRD to cluster
   kubectl apply -f config/crd/bases/net.futhwo.io_externalbalancers.yaml
   
   # Run operator locally
   go run ./main.go
   
   # Apply sample CR
   kubectl apply -f config/samples/net_v1alpha1_externalbalancer_single.yaml
   ```

2. **Requirements**:
   - Go 1.22+
   - Kubernetes 1.26+
   - Traefik CRDs installed in cluster

## Key Files for Understanding
- `api/v1alpha1/externalbalancer_types.go` - Core API types and validation
- `controllers/externalbalancer_controller.go` - Main reconciliation logic
- `config/samples/*.yaml` - Example usage patterns

## Project-Specific Conventions
1. **Error Handling**: All errors from reconciliation are returned to trigger requeuing
2. **Validation**: Port consistency is enforced for SingleService strategy
3. **Labels**: IngressRoute-specific labels supported via `ingressRouteLabels` field