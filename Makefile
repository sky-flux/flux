.PHONY: all test cover lint bench vet examples

all: vet lint test

test:
	go test ./... -cover

cover:
	go test -coverprofile=coverage.txt ./...
	go tool cover -func=coverage.txt

lint:
	golangci-lint run ./...

bench:
	go test -bench=. -benchmem -run='^$$' ./... -timeout=120s

vet:
	go vet ./...

examples:
	go build -o /dev/null ./examples/basic/
	go build -o /dev/null ./examples/optimizer/
	go build -o /dev/null ./examples/reschedule/
