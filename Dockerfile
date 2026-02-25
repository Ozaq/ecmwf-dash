FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN go build -ldflags="-s -w" -o /ecmwf-dash cmd/server/main.go
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /healthcheck cmd/healthcheck/main.go

FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=builder --chown=nobody:nobody /ecmwf-dash /ecmwf-dash
COPY --from=builder --chown=nobody:nobody /healthcheck /healthcheck
COPY --chown=nobody:nobody web ./web
USER nobody
EXPOSE 8000
HEALTHCHECK --interval=30s --timeout=5s CMD ["/healthcheck"]
ENTRYPOINT ["./ecmwf-dash"]
