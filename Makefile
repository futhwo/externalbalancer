.PHONY: run
run:
	GO111MODULE=on go run ./main.go

.PHONY: crd
crd:
	kubectl apply -f config/crd/bases/net.futhwo.io_externalbalancers.yaml

.PHONY: samples
samples:
	kubectl apply -f config/samples/net_v1alpha1_externalbalancer_weighted.yaml
	kubectl apply -f config/samples/net_v1alpha1_externalbalancer_single.yaml
