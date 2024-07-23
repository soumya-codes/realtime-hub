.PHONY: clean-images images setup teardown manage-dependencies sync-workspace

# Define image names or tags
HUBSERVER_IMAGE = hubserver
HUBCLIENT_IMAGE = hubclient

# Target to clean up specific Docker images
clean-images:
	docker rmi -f $(HUBSERVER_IMAGE) $(HUBCLIENT_IMAGE)

images: clean-images
	@echo "Building hubserver..."
	docker build -t hubserver:latest -f hubserver/build/Dockerfile ./hubserver
	@echo "Building hubclient..."
	docker build -t hubclient:latest -f hubclient/build/Dockerfile ./hubclient


setup: teardown images
	docker-compose -f run/docker-compose.yaml up -d

teardown:
	docker-compose -f run/docker-compose.yaml down --remove-orphans

manage-dependencies:
	@echo "Tidying, downloading, and verifying dependencies for hubserver..."
	cd hubserver && go mod tidy && go mod download && go mod verify
	@echo "Tidying, downloading, and verifying dependencies for hubclient..."
	cd hubclient && go mod tidy && go mod download && go mod verify

sync-workspace:
	@echo "Syncing go mod directories..."
	go work sync