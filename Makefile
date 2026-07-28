FRONTEND_IMAGE = infraunnes/gpu-server-monitoring-frontend

.PHONY: build-agent build-frontend push-frontend clean

build-agent:
	docker build -f Dockerfile.agent -t gpu-monitoring-agent-builder .
	CID=$$(docker create gpu-monitoring-agent-builder) && \
	docker cp $$CID:/gpu-monitoring-agent-linux-amd64 ./release/gpu-monitoring-agent-linux-amd64 && \
	docker cp $$CID:/gpu-monitoring-agent-linux-arm64 ./release/gpu-monitoring-agent-linux-arm64 && \
	docker rm $$CID
	chmod +x ./release/gpu-monitoring-agent-*
	@echo " binaries in release/"
	@ls -lh release/

build-frontend:
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		-f Dockerfile.frontend \
		-t $(FRONTEND_IMAGE):latest \
		--push \
		.

push-frontend:
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		-f Dockerfile.frontend \
		-t $(FRONTEND_IMAGE):latest \
		--push \
		.

clean:
	rm -rf release/
