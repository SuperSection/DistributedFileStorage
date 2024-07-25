build:
	@go build -o bin/filestorage

run: build
	@./bin/filestorage

test:
	@go test ./... -v --race 