build: test-all
	go build -v ./flyte

test-all:
	go test ./... -tags="acceptance"

test:
	go test ./...

docs:
	godoc -http=:6060