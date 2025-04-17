# Linear Clock
Tested on Raspberry Pi Zero 2


## Setup

1. Setup headless Pi
2. MUST BE 32bit!!!


2. Transfer files
on pi
 - `mkdir -p /home/pi/go/github.com/jaredwarren/clock`

on mac

`scp config-armv7 pi@clock.local:/home/pi/go/github.com/jaredwarren/clock/config-armv7`
`scp clock/clock-armv7 pi@clock.local:/home/pi/go/github.com/jaredwarren/clock/clock-armv7`

`scp -r templates pi@clock.local:/home/pi/go/github.com/jaredwarren/clock`


1. Setup systemd
### setup config server
`sudo nano /lib/systemd/system/config.service`
paste config.service

or

`rsync --rsync-path="sudo rsync" config.service pi@clock.local:/lib/systemd/system/config.service`



`sudo systemctl restart config.service` 

### setup clock
`sudo nano /lib/systemd/system/clock.service`
paste clock.service

or

`rsync --rsync-path="sudo rsync" config.service pi@clock.local:/lib/systemd/system/config.service`


`sudo systemctl restart clock.service`




- - - - - - - - - - - - - - - - - - - - - - - - - - - - - - 
# Build and update clock
## build docker builder image on mac
only once or when updating dockerfile
`docker buildx build --platform linux/arm/v7 --tag ws2811-builder --file docker/app-builder/Dockerfile .`

## use image to cross compile app on mac for pi
`APP=clock`̋̊̋
`docker run --rm -v "$PWD":/usr/src/$$APP --platform linux/arm/v7 -w /usr/src/$APP ws2811-builder:latest go build -a -o "$APP-armv7" -v`

### on pi
`sudo systemctl stop clock.service`

## on mac push app to pi
`scp clock-armv7 pi@clock.local:/home/pi/go/github.com/jaredwarren/clock/clock-armv7`

### on pi
`sudo systemctl restart clock.service`


- - - - - - - - - - - - - - - - - - - - - - - - - - - - - - 

# Build & update config server
build config server on mac
`GOOS=linux GOARCH=arm GOARM=7 go build -o config-armv7 cmd/server/main.go`

## Push to pi
### on pi
`sudo systemctl stop config.service`

### on mac
`scp config-armv7 pi@clock.local:/home/pi/go/github.com/jaredwarren/clock/config-armv7`
push config
`scp -r templates pi@clock.local:/home/pi/go/github.com/jaredwarren/clock`

### on Pi
`sudo systemctl restart config.service`

test
`http://clock.local:8080`





 



 `sudo systemctl restart player.service`  or `sudo reboot now` 




# TODO #
 - fix test
  - colors
  - icon
- resresh too slow
- make sure brightness config works
- 


