FROM golang:1.24.2-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -mod=vendor -o sni-proxy

FROM alpine:3.22.0
WORKDIR /app
COPY --from=builder /app/sni-proxy .
ENTRYPOINT ["./sni-proxy"]
