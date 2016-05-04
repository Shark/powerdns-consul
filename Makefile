build:
	go build .

test:
	go test ./...

end2end_test:
	end2end_test/run

.PHONY: build test end2end_test
