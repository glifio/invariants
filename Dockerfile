# syntax=docker/dockerfile:1
FROM golang:1.25-bookworm AS build

ENV CGO_ENABLED=0

WORKDIR /build
COPY . .

RUN go build -o /build/invariants ./cmd/invariants

FROM debian:bookworm-slim

RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates \
 && rm -rf /var/lib/apt/lists/*

RUN update-ca-certificates

COPY --from=build /build/invariants /app/invariants

WORKDIR /app

CMD ["/app/invariants", "monitor"]
