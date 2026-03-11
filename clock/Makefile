.EXPORT_ALL_VARIABLES:
APP = clockd
ROOT = ..

# Variables for deployment locally
# pi@clock.local
USER = pi
HOST = clock.local

.PHONY: build
build: check-builder-image
	docker run --rm -v "$$(cd $(ROOT) && pwd)":/src --platform linux/arm/v7 -w /src ws2811-builder:latest go build -a -o "$$APP-armv7" -v ./cmd/clockd

.PHONY: check-builder-image
check-builder-image:
	@if [ -z "$(shell docker images -q ws2811-builder:latest)" ]; then \
		$(MAKE) build-image; \
	fi

.PHONY: build-image
build-image:
	docker buildx build --platform linux/arm/v7 --tag ws2811-builder --file docker/app-builder/Dockerfile $(ROOT)


.PHONY: clean
clean:
	@echo "rm -f $(ROOT)/$$APP-armv7"
	@rm -f $(ROOT)/$$APP-armv7

.PHONY: push
push:
	scp $(ROOT)/$$APP-armv7 $$USER@$$HOST:/home/pi/go/github.com/jaredwarren/clock/$$APP-armv7