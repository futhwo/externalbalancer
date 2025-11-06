# externalbalancer (fixed)

Same as before, but with manual DeepCopy implementations (so you don't need codegen)
and controller-runtime bumped to v0.18.4 to match k8s 0.30.x.

Quick start:
1) `kubectl apply -f config/crd/bases/net.cdlan.io_externalbalancers.yaml`
2) `go run ./main.go`
3) Apply one of the samples in `config/samples/`.
