.EXPORT_ALL_VARIABLES:
APP = clock

# Variables for deployment locally
# pi@clock.local
USER = pi
HOST = clock.local

.PHONY: gen
gen:
	go generate ./...

.PHONY: build
build: gen check-builder-image
	docker run --rm -v "$$PWD":/usr/src/$$APP --platform linux/arm/v7 -w /usr/src/$$APP ws2811-builder:latest go build -a -o "$$APP-armv7" -v

.PHONY: check-builder-image
check-builder-image:
	@if [ -z "$(shell docker images -q ws2811-builder:latest)" ]; then \
		$(MAKE) build-image; \
	fi

.PHONY: build-image
build-image:
	docker buildx build --platform linux/arm/v7 --tag ws2811-builder --file docker/app-builder/Dockerfile .


.PHONY: clean
clean:
	@echo "rm -f $$APP-armv7"
	@rm -f $$APP-armv7

.PHONY: push
push: 
	scp $$APP-armv7 $$USER@$HOST:/home/pi/go/github.com/jaredwarren/clock/$$APP-armv7