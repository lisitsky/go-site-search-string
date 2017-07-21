all: get test

get:
	go get -v -u -t github.com/lisitsky/go-site-search-string

install:
	go install

test:
	HTTP_TIMEOUT=1 go test -v -cover -race

build:
	go build -v -o build/go-site-search-string

container: build
	docker build -t lisitsky/go-site-search-string .