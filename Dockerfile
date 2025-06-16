FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY cmd ./cmd
COPY internal ./internal
COPY go.mod ./
COPY go.sum ./
RUN go mod download
RUN go build -o /ecmwf-dash cmd/server/main.go
FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=builder --chown=nobody:nobody /ecmwf-dash /ecmwf-dash
COPY web ./web
COPY config.yaml ./
USER nobody
EXPOSE 8000
ENTRYPOINT ["./ecmwf-dash"]
