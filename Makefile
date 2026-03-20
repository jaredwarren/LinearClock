.EXPORT_ALL_VARIABLES:
# Binaries: clockd (LED clock), configd (config HTTP server)
USER = pi
HOST = clock.local
# SSH user for deploy script (script uses DEPLOY_USER to avoid shell USER)
DEPLOY_USER = pi

.PHONY: run-configd run-clockd
run-configd:
	go run cmd/configd/main.go

run-clockd:
	go run cmd/cli-clock/main.go

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
	rm -rf dist
	rm -f clockd configd clockd-armv7 configd-armv7

.PHONY: push
push: build-clockd-arm build-configd-arm
	scp clockd-armv7 configd-armv7 $$USER@$$HOST:/home/pi/go/github.com/jaredwarren/clock/

# Deploy: build, rsync to Pi, restart systemd (see scripts/deploy.sh --help)
.PHONY: deploy deploy-clock deploy-config
deploy:
	./scripts/deploy.sh both
deploy-clock:
	./scripts/deploy.sh clock
deploy-config:
	./scripts/deploy.sh config

.PHONY: test
test:
	go test ./...

.PHONY: verify
verify:
	go test ./internal/config ./internal/calendar ./internal/display ./internal/server
	go test ./internal/server -run TestTemplateParseSmoke -count=1
	go vet ./cmd/configd ./cmd/cli-clock ./internal/config ./internal/calendar ./internal/display ./internal/server

.PHONY: release-dist
release-dist: verify build-clockd-arm build-configd-arm
	mkdir -p dist
	cp clockd-armv7 dist/
	cp configd-armv7 dist/
	cp clock.service dist/
	cp config.service dist/
	cp -R templates dist/
	cp -R public dist/
	(cd dist && shasum -a 256 clockd-armv7 configd-armv7 > SHA256SUMS)
