build:
	go fmt ./...
	go vet ./...
	go build -o mchfuse main.go

test:
	go test ./...

lint:
	golangci-lint run --fix

clean:
	rm -f mchfuse
