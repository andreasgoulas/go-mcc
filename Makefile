TARGET   ?= go-mcc
CORE_OUT ?= plugins/core.so
GO       ?= go
GOFLAGS  ?=

all: build build_core

build:
	$(GO) $(GOFLAGS) build -o $(TARGET) .

build_core:
	@mkdir -p plugins
	$(GO) $(GOFLAGS) build -buildmode=plugin -o $(CORE_OUT) ./core

clean:
	rm $(TARGET) $(CORE_OUT)

fmt:
	$(GO) fmt ./core ./mcc .

.PHONY: all build build_core clean fmt
