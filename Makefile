all: get test

get:
	go get -v -u -t github.com/lisitsky/go-site-search-string

install:
	go install

test:
	HTTP_TIMEOUT=1 go test -v -cover -race