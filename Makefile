all: get install

get:
	go get -v -u github.com/lisitsky/go-site-search-string

install:
	go install

test:
	HTTP_TIMEOUT=1 go test -v -cover -race