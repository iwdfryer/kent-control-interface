SHELL := /bin/bash

CUR_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

.PHONY = setup binaries clean

all: setup binaries

setup:
	$(shell [ ! -d "internal" ] && ln -s "${GOPATH}/src/dk-srv/internal" internal)
	$(shell [ ! -f "wasm_exec.js" ] && cp "${GOROOT}/misc/wasm/wasm_exec.js" $(CUR_DIR))

binaries:
	go build -o ws-kent ws-kent.go
	env GOOS=linux GOARCH=arm GOARM=5 go build -o ws-kent-pi ws-kent.go
	GOARCH=wasm GOOS=js go build -o lib.wasm wasm.go

clean:
	-rm internal
	-rm wasm_exec.js
	-rm ws-kent
	-rm lib.wasm
