.PHONY: \
	all \
	deps \
	update-deps \
	test-deps \
	update-test-deps \
	build \
	install \
	lint \
	vet \
	errcheck \
	pretest \
	test \
	cov \
	clean

all: test

deps:
	go get -d -v ./...

update-deps:
	go get -d -v -u -f ./...

test-deps:
	go get -d -v -t ./...

update-test-deps:
	go get -d -v -t -u -f ./...

build: deps
	go build ./...

install: deps
	go install ./...

lint: testdeps
	go get -v github.com/golang/lint/golint
	golint ./.

vet: testdeps
	go get -v golang.org/x/tools/cmd/vet
	go vet ./...

errcheck: testdeps
	go get -v github.com/kisielk/errcheck
	errcheck ./...

pretest: lint vet errcheck

test: testdeps pretest
	go test -test.v ./...

cov: testdeps
	go get -v github.com/axw/gocov/gocov
	go get golang.org/x/tools/cmd/cover
	gocov test | gocov report

clean:
	go clean -i ./...
