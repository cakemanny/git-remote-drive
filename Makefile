
.PHONY: lint
lint:
	golangci-lint run


.PHONY: update-deps
update-deps:
	go get -v -u
