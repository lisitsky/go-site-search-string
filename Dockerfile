###
# Basic image builder
FROM golang:onbuild AS builder
#EXPOSE 8080



###
#
#FROM builder




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
