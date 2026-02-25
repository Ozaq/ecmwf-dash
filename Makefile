.PHONY: build test vet run docker-build docker-run clean

build:
	go build -o ecmwf-dash cmd/server/main.go

test:
	go test -race ./...

vet:
	go vet ./...

run: build
	./ecmwf-dash

docker-build:
	docker build -t ecmwf-dash .

docker-run:
	docker run -e GITHUB_TOKEN=$(GITHUB_TOKEN) -v $(PWD)/config.yaml:/config.yaml -p 8000:8000 ecmwf-dash

clean:
	rm -f ecmwf-dash
