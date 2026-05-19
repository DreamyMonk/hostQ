FROM golang:1.22-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/hostq-panel .

FROM alpine:3.20
RUN adduser -D -u 10001 hostq
WORKDIR /app
COPY --from=builder /out/hostq-panel /usr/local/bin/hostq-panel
ENV HOSTQ_ADDR=0.0.0.0:8090
ENV HOSTQ_DATA_DIR=/data
ENV WEB_ROOT=/var/www
EXPOSE 8090
VOLUME ["/data", "/var/www"]
CMD ["/usr/local/bin/hostq-panel"]
