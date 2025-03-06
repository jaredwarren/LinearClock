.EXPORT_ALL_VARIABLES:
APP = clock

build:
	docker run --rm -v "$$PWD":/usr/src/$$APP --platform linux/arm/v7 -w /usr/src/$$APP ws2811-builder:latest go build -a -o "$$APP-armv7" -v

push: 
	scp $$APP-armv7 pi@clock.local:/home/pi/go/github.com/jaredwarren/clock/$$APP-armv7
