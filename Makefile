.PHONY: build test vet lint validate verify clean

build:
	go build -o bin/maslow ./cmd/maslow/

test:
	go test ./...

vet:
	go vet ./...

lint: vet
	cue vet schema/maslow.cue

validate: build
	bin/maslow validate maslow.yaml

verify: build
	bin/maslow verify --profile full

clean:
	rm -rf bin/ reports/verify.json
