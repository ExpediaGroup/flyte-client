test:
	go test ./...
acceptance:
	go test ./... -tags=acceptance