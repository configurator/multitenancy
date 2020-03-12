# Go options
GO111MODULE ?= auto
CGO_ENABLED ?= 0

# Operator SDK options
SDK_VERSION ?= v0.15.2
OPERATOR_SDK ?= _bin/operator-sdk

# Detect platform for operator-sdk (only supports linux/macOS)
UNAME := $(shell uname)
ifeq ($(UNAME), Linux)
SDK_PLATFORM := linux-gnu
GOROOT ?= /usr/lib/go  # This is the location on arch linux, override in environment if needed
endif
ifeq ($(UNAME), Darwin)
SDK_PLATFORM := apple-darwin
GOROOT ?= /usr/local/opt/go/libexec
endif

# Download URL for Operator SDK based off version and platform
OPERATOR_SDK_DOWNLOAD_URL ?= https://github.com/operator-framework/operator-sdk/releases/download/${SDK_VERSION}/operator-sdk-${SDK_VERSION}-x86_64-${SDK_PLATFORM}

# Image Options
IMAGE_TAG ?= configurator/multitenancy-controller

# Build the operator docker image
.PHONY: build
LD_FLAGS ?= -ldflags -X=github.com/configurator/multitenancy/version.CommitSHA=`git rev-parse HEAD`
build: ${OPERATOR_SDK}
	CGO_ENABLED=${CGO_ENABLED} GOOS=linux ${OPERATOR_SDK} build ${IMAGE_TAG} --go-build-args "${LD_FLAGS}"

# Pushes the docker image to a registry
push: build
	docker push ${IMAGE_TAG}

# Ensures a local copy of the operator-sdk
${OPERATOR_SDK}:
	mkdir -p $(dir ${OPERATOR_SDK})
	curl -JL -o ${OPERATOR_SDK} ${OPERATOR_SDK_DOWNLOAD_URL}
	chmod +x ${OPERATOR_SDK}

# Generates deep copy code
generate: ${OPERATOR_SDK}
	GOROOT=${GOROOT} ${OPERATOR_SDK} generate k8s

# Generates CRD manifest
manifests: ${OPERATOR_SDK}
	${OPERATOR_SDK} generate crds

##
# Kind helpers for local testing
##
KIND ?= _bin/kind
KIND_VERSION ?= v0.7.0
KIND_DOWNLOAD_URL ?= https://github.com/kubernetes-sigs/kind/releases/download/${KIND_VERSION}/kind-$(shell uname)-amd64
CLUSTER_NAME ?= multitenancy

# Ensures a repo-local installation of kind
${KIND}:
	mkdir -p $(dir ${KIND})
	curl -JL -o ${KIND} ${KIND_DOWNLOAD_URL}
	chmod +x ${KIND}

# This workflow could be adapted for testing in pipelines
acctest: ${KIND} cluster load deploy
	                                   # an acceptance test against the local cluster
																		 # could go here

# Create a local cluster for testing
cluster: ${KIND}
	${KIND} create cluster --name ${CLUSTER_NAME}

# Load the docker image into the local kind cluster (required unless pushing the image to a registry first)
load: ${KIND} build
	${KIND} load docker-image --name ${CLUSTER_NAME} ${IMAGE_TAG}

# Delete the local kind cluster
del_cluster: ${KIND}
	${KIND} delete cluster --name ${CLUSTER_NAME}

# Deploys the helm chart with default options to the current kube context.
# When using kind and the `make cluster` target above - your kubectl context will
# automatically be set to the kind cluster when it's done provisioning.
.PHONY: deploy
deploy:
	helm install multitenancy deploy/charts/multitenancy --set image.repository=${IMAGE_TAG}

# Apply the example charts
examples:
	kubectl apply -f deploy/example/multitenancy.yaml
	kubectl apply -f deploy/example/tenants.yaml
