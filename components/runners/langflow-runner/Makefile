IMAGE_NAME ?= vteam-langflow-runner
REGISTRY ?= quay.io/ambient_code
TAG ?= latest
CONTAINER_ENGINE ?= docker

.PHONY: build
build:
	cd .. && $(CONTAINER_ENGINE) build -t $(IMAGE_NAME):$(TAG) -f langflow-runner/Dockerfile .

.PHONY: push
push: build
	$(CONTAINER_ENGINE) tag $(IMAGE_NAME):$(TAG) $(REGISTRY)/$(IMAGE_NAME):$(TAG)
	$(CONTAINER_ENGINE) push $(REGISTRY)/$(IMAGE_NAME):$(TAG)

.PHONY: load-minikube
load-minikube: build
	$(CONTAINER_ENGINE) save localhost/$(IMAGE_NAME):$(TAG) -o /tmp/langflow-runner.tar || \
		$(CONTAINER_ENGINE) save $(IMAGE_NAME):$(TAG) -o /tmp/langflow-runner.tar
	minikube image load /tmp/langflow-runner.tar
	minikube ssh "docker tag localhost/$(IMAGE_NAME):$(TAG) $(IMAGE_NAME):$(TAG) || docker tag $(IMAGE_NAME):$(TAG) $(IMAGE_NAME):$(TAG)"
	rm /tmp/langflow-runner.tar

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build           - Build the langflow-runner image"
	@echo "  push            - Build and push to registry"
	@echo "  load-minikube   - Build and load into minikube"
	@echo ""
	@echo "Variables:"
	@echo "  CONTAINER_ENGINE - Container build tool (default: docker, or: podman)"
	@echo "  TAG             - Image tag (default: latest)"
	@echo "  REGISTRY        - Container registry (default: quay.io/ambient_code)"
