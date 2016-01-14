PROG=hearth
OUTDIR=bin

default: build

deps:
	@go get ./...
	@mkdir -p $(OUTDIR)

build: fmt
	@go build -o $(OUTDIR)/$(PROG)

clean:
	@rm -rf $(OUTDIR)

run: build
	@$(OUTDIR)/$(PROG)

fmt:
	@go fmt ./...

test: fmt
	@go test ./...
