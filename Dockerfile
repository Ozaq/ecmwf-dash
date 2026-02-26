FROM golang:1.26-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.Version=${VERSION}" -o /ecmwf-dash cmd/server/main.go
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /healthcheck cmd/healthcheck/main.go

FROM gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=builder --chown=nobody:nobody /ecmwf-dash /ecmwf-dash
COPY --from=builder --chown=nobody:nobody /healthcheck /healthcheck
COPY --chown=nobody:nobody web ./web
USER nobody
EXPOSE 8000
HEALTHCHECK --interval=30s --timeout=5s CMD ["/healthcheck"]
ENTRYPOINT ["./ecmwf-dash"]
