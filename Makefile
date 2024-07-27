help:
	@echo "Usage:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ":" | sed -e 's/^/  /'

## build-cli: build the cli application
.PHONY: build-cli
build-cli:
	go build -o env ./main.go