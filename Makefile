all:
	@echo "nothing is done when 'all' is done, try make help"
help:	## show this help
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'
.PHONY: test
test:	export QSTF_TEST_CACHE = 1
test:	## go test the application
	go test ./...
.PHONY: test-again
test-again:	## go test the application again
	go test -count=1 ./...
.PHONY: build
build: build-all build-test	## go build the application
.PHONY: build-all
build-all:	## go build all
	go build ./...
.PHONY: build-test
build-test:	## go build the test application
	go build -o build/test cmd/test/main.go