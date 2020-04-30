
ifeq ($(PREFIX),)
    PREFIX := /usr/local
endif

.PHONY: all
all: mchfuse

mchfuse:
	go fmt ./...
	go vet ./...
	go build -o mchfuse main.go

.PHONY: test
test:
	go test ./...

.PHONY: lint
lint:
	golangci-lint run --fix

.PHONY: clean
clean:
	rm -f mchfuse

.PHONY: install
install: mchfuse
	install -d $(DESTDIR)$(PREFIX)/bin/
	install -m 755 mchfuse $(DESTDIR)$(PREFIX)/bin/

.PHONY: uninstall
uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/mchfuse
