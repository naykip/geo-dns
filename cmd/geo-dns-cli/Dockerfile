FROM golang:1.21-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o geo-dns-cli .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /build/geo-dns-cli /usr/local/bin/geo-dns-cli
ENTRYPOINT ["geo-dns-cli"]
