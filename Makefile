.PHONY: run
run:
	GO111MODULE=on go run ./main.go

.PHONY: build
build:
	GO111MODULE=on go build -o bin/manager main.go

.PHONY: crd
crd:
	kubectl apply -f config/crd/bases/net.futhwo.io_externalbalancers.yaml

.PHONY: samples-single
samples-single:
	kubectl apply -f config/samples/net_v1alpha1_externalbalancer_single.yaml

.PHONY: samples-wrr
samples-wrr:
	kubectl apply -f config/samples/net_v1alpha1_externalbalancer_weighted.yaml

