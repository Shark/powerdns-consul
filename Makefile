build:
	go build .

docker:
	GOOS=linux GOARCH=amd64 go build .
	docker build --force-rm -t sh4rk/powerdns-consul .
	docker push sh4rk/powerdns-consul

test:
	go test ./...

end2end_test:
	end2end_test/run

.PHONY: build test end2end_test
