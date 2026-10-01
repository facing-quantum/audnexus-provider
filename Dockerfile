# syntax=docker/dockerfile:1
FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -ldflags="-s -w" -o /audnexus-provider ./cmd/server

# Final minimal runtime image
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -S audnexus && adduser -S audnexus -G audnexus \
    && mkdir -p /cache && chown -R audnexus:audnexus /cache

WORKDIR /app
COPY --from=builder /audnexus-provider /app/audnexus-provider

USER audnexus

ENV PORT=8080 \
    REGION=us \
    CACHE_DIR=/cache \
    LOG_LEVEL=INFO

EXPOSE 8080
VOLUME ["/cache"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/audnexus-provider"]
