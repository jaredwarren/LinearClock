# Stage 0 : Build the C library

FROM debian AS lib_builder

WORKDIR /foundry

RUN apt-get update -y && apt-get install -y \
  build-essential \
  cmake \
  git

RUN git clone https://github.com/jgarff/rpi_ws281x.git \
  && cd rpi_ws281x \
  && mkdir build \
  && cd build \
  && cmake -D BUILD_SHARED=OFF -D BUILD_TEST=OFF .. \
  && cmake --build . \
  && make install

# Stage 1 : Build a go image with the rpi_ws281x C library

FROM golang:latest
COPY --from=lib_builder /usr/local/lib/libws2811.a /usr/local/lib/
COPY --from=lib_builder /usr/local/include/ws2811 /usr/local/include/ws2811

# needed to fix `github.com/rpi-ws281x/rpi-ws281x-go@v1.0.10: Get "https://proxy.golang.org/github.com/rpi-ws281x/rpi-ws281x-go/@v/v1.0.10.zip": net/http: TLS handshake timeout`
ENV GOPROXY=direct

WORKDIR /go/src/rpi-ws281x-go

COPY . ./
# COPY go.mod .
# COPY go.sum .
# COPY *.go ./