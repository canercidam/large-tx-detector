export GOBIN = $(shell pwd)/toolbin

ABIGEN = $(GOBIN)/abigen

.PHONY: tools
tools:
	@rm -rf toolbin
	@go install github.com/ethereum/go-ethereum/cmd/abigen

.PHONY: gen
gen:
	@$(ABIGEN) --sol ./contracts/IERC20.sol --out ./contracts/IERC20.go --pkg contracts --type IERC20

.PHONY: run
run:
	@mkdir -p build
	@go build -o build/app
	@./build/app

.PHONY: docker-build
docker-build:
	@docker build --no-cache --tag largetx:latest .

.PHONY: docker
docker:
	@docker run -p 8080:8080 --env-file .env -it largetx:latest
