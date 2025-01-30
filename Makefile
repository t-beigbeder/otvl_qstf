all:
	@echo "nothing is done when 'all' is done, try make help"
help:	## show this help
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'
.PHONY: test
test:	## go test the application
	go test ./...
.PHONY: build
build:	## go build the application
	go build ./...