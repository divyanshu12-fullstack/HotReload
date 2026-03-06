.PHONY: build run demo test clean

GO ?= go
GOEXE := $(shell $(GO) env GOEXE)
HOTRELOAD_BIN := ./bin/hotreload$(GOEXE)
TESTSERVER_BIN := ./bin/testserver$(GOEXE)
GOTMPDIR := $(CURDIR)/.gotmp

build:
	$(GO) build -o $(HOTRELOAD_BIN) .

run: build
	$(HOTRELOAD_BIN) \
		--root ./testserver \
		--build "$(GO) build -o $(TESTSERVER_BIN) ./testserver" \
		--exec "$(TESTSERVER_BIN)"

demo: run

test:
	mkdir -p $(GOTMPDIR)
	GOTMPDIR="$(GOTMPDIR)" $(GO) test ./...

clean:
	rm -rf ./bin
	rm -rf $(GOTMPDIR)
