# Image URL to use all building/pushing image targets
IMG ?= docker.io/moulick/ingress-whitelister:latest
COVERAGE_FILE ?= cover.out
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec


##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
YQ ?= $(LOCALBIN)/yq
GOJQ ?= $(LOCALBIN)/gojq
GINKGO ?= $(LOCALBIN)/ginkgo
JSONNET ?= $(LOCALBIN)/jsonnet
JSONNET_FMT ?= $(LOCALBIN)/jsonnetfmt

## Tool Versions
ENVTEST_K8S_VERSION = 1.24.2
KUSTOMIZE_VERSION ?= v4.5.7
CONTROLLER_GEN_VERSION ?= v0.7.0
JSONNET_VERSION ?= v0.19.1
YQ_VERSION ?= v4.30.8
GINKGO_VERSION ?= v2.8.0
GOJQ_VERSION ?= v0.12.11

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest ginkgo ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" $(GINKGO) -r --trace --race --randomize-all --randomize-suites --fail-fast --coverprofile=${COVERAGE_FILE}

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

.PHONY: docker-all
docker-all: docker-build docker-push ## Build and push docker image with the manager.

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

bin: ## Create a bin folder for temp storage.
	mkdir -p bin

CRD_OUT = bin/crds.yaml
.PHONY: crds
crds: manifests kustomize ## Generate CRDs into the bin directory.
	$(KUSTOMIZE) build config/crd > $(CRD_OUT)

.PHONY: jsonnet-crd
jsonnet-crd: yq gojq jsonnet crds ## Generate CRDs in the form of jsonnet files.
	@$(YQ) eval -I=0 $(CRD_OUT) -o=json | $(GOJQ) -s . | $(JSONNET) - | $(JSONNET_FMT) --max-blank-lines 1 - -o jsonnet/crds.libsonnet

.PHONY: bundle
bundle: jsonnet-crd ## Generate deployment files into the bin directory.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default > bin/bundle.yaml

.PHONY: install
install: bin crds ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	kubectl apply -f $(CRD_OUT)

.PHONY: uninstall
uninstall: crds ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	kubectl delete --ignore-not-found=$(ignore-not-found) -f $(CRD_OUT)

.PHONY: deploy
deploy: bundle install ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	kubectl apply -f bin/manifests_out.yaml

.PHONY: undeploy
undeploy: bundle uninstall ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	kubectl delete --ignore-not-found=$(ignore-not-found) -f bin/manifests_out.yaml

##@ Install Dependencies

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	@test -s $(KUSTOMIZE) || { curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: controller-gen
controller-gen: $(LOCALBIN) ## Download controller-gen locally if necessary.
	@if test -s $(CONTROLLER_GEN); then \
		if [ $(CONTROLLER_GEN_VERSION) = $(word 2,$(shell $(CONTROLLER_GEN) --version)) ]; then \
			echo "Correct version of controller-gen is already installed"; \
		else \
			echo "Wrong version of controller-gen is installed, reinstalling version $(CONTROLLER_GEN_VERSION)"; \
			GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION); \
			echo "controller-gen installed"; \
		fi \
	else \
		echo "controller-gen not installed, installing version $(CONTROLLER_GEN_VERSION)"; \
		GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION); \
		echo "controller-gen installed"; \
	fi

.PHONY: jsonnet
jsonnet: $(LOCALBIN) ## Download jsonnet locally if necessary.
	@if test -s $(JSONNET); then \
		if [ $(JSONNET_VERSION) = $(word 6,$(shell $(JSONNET) --version)) ]; then \
			echo "Correct version of Jsonnet is already installed"; \
		else \
			echo "Wrong version of Jsonnet is installed, reinstalling version $(JSONNET_VERSION)"; \
			GOBIN=$(LOCALBIN) go install github.com/google/go-jsonnet/cmd/...@$(JSONNET_VERSION); \
			echo "Jsonnet installed"; \
		fi \
	else \
		echo "Jsonnet not installed, installing version $(JSONNET_VERSION)"; \
		GOBIN=$(LOCALBIN) go install github.com/google/go-jsonnet/cmd/...@$(JSONNET_VERSION); \
		echo "Jsonnet installed"; \
	fi

.PHONY: yq
yq: $(LOCALBIN) ## Download yq locally if necessary.
	@if test -s $(YQ); then \
		if [ $(YQ_VERSION) = $(word 4,$(shell $(YQ) --version)) ]; then \
			echo "Correct version of yq is already installed"; \
		else \
			echo "Wrong version of yq is installed, reinstalling version $(YQ_VERSION)"; \
			GOBIN=$(shell pwd)/bin go install github.com/mikefarah/yq/v4@$(YQ_VERSION); \
			echo "yq installed"; \
		fi \
	else \
		echo "yq not installed, installing version $(YQ_VERSION)"; \
		GOBIN=$(shell pwd)/bin go install github.com/mikefarah/yq/v4@$(YQ_VERSION); \
		echo "yq installed"; \
	fi

.PHONY: gojq
gojq: $(LOCALBIN) ## Download gojq locally if necessary.
	@if test -s $(JQ); then \
		if [ $(GOJQ_VERSION) = v$(word 2,$(shell $(GOJQ) --version)) ]; then \
			echo "Correct version of gojq is already installed"; \
		else \
			echo "Wrong version of gojq is installed, reinstalling version $(GOJQ_VERSION)"; \
			GOBIN=$(shell pwd)/bin go install github.com/itchyny/gojq/cmd/gojq@$(GOJQ_VERSION); \
			echo "gojq installed"; \
		fi \
	else \
		echo "gojq not installed, installing version $(YQ_VERSION)"; \
		GOBIN=$(shell pwd)/bin go install github.com/itchyny/gojq/cmd/gojq@$(GOJQ_VERSION); \
		echo "gojq installed"; \
	fi

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	@test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: ginkgo
ginkgo: $(LOCALBIN) ## Download ginkgo locally if necessary.
	@if test -s $(GINKGO); then \
		if [ $(GINKGO_VERSION) = v$(word 3,$(shell $(GINKGO) version)) ]; then \
			echo "Correct version of Ginkgo is already installed"; \
		else \
			echo "Wrong version of Ginkgo is installed, reinstalling new"; \
			GOBIN=$(LOCALBIN) go install github.com/onsi/ginkgo/v2/ginkgo@$(GINKGO_VERSION); \
			echo "Ginkgo installed"; \
		fi \
	else \
		echo "Ginkgo not installed, installing version $(GINKGO_VERSION)"; \
		GOBIN=$(LOCALBIN) go install github.com/onsi/ginkgo/v2/ginkgo@$(GINKGO_VERSION); \
		echo "Ginkgo installed"; \
	fi
