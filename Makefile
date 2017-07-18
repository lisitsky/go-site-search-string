all: install

install:
	go install

test:
	HTTP_TIMEOUT=1 go test -v -cover -race