###
# Basic image builder
FROM golang:1.8  AS builder

WORKDIR /Users/el/go/src/github.com/lisitsky/go-site-search-string

RUN go get -u -v -t github.com/lisitsky/go-site-search-string

COPY main.go .

RUN go build -v -o go-site-search-string .

EXPOSE 8080



###
#
FROM builder

WORKDIR /go/

COPY --from=builder /go/src/github.com/lisitsky/go-site-search-string .

CMD ["/go-site-search-string"]




#FROM golang:1.8
#
#RUN set -x && \
#	go get -u -v -t github.com/lisitsky/go-site-search-string && \
#	HTTP_TIMEOUT=1 go test -v -cover -race	&& \
#	go build github.com/lisitsky/go-site-search-string
#
#ADD ./go-site-search-string /go-site-search-string
#
#CMD ["/go-site-search-string"]
