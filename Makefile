.EXPORT_ALL_VARIABLES:
# Binaries: clockd (LED clock), configd (config HTTP server)
USER = pi
HOST = clock.local

.PHONY: build-clockd build-configd build-all
build-clockd:
	$(MAKE) -C clock build

build-configd:
	go build -o configd -v ./cmd/configd

build-all: build-clockd build-configd

# Cross-build clockd for ARM: delegates to clock/Makefile
.PHONY: build-clockd-arm
build-clockd-arm:
	$(MAKE) -C clock build

build-configd-arm:
	GOOS=linux GOARCH=arm GOARM=7 go build -o configd-armv7 -v ./cmd/configd

.PHONY: clean
clean:
	rm -f clockd configd clockd-armv7 configd-armv7

.PHONY: push
push: build-clockd-arm build-configd-arm
	scp clockd-armv7 configd-armv7 $$USER@$$HOST:/home/pi/go/github.com/jaredwarren/clock/

.PHONY: test
test:
	go test ./...
