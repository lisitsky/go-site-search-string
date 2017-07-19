# Go Lang  test

[![Build Status](https://travis-ci.org/lisitsky/go-site-search-string.svg?branch=master)](https://travis-ci.org/lisitsky/go-site-search-string)  [![Coverage Status](https://coveralls.io/repos/github/lisitsky/go-site-search-string/badge.svg?branch=master)](https://coveralls.io/github/lisitsky/go-site-search-string?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/lisitsky/go-site-search-string)](https://goreportcard.com/report/github.com/lisitsky/go-site-search-string)  [![GoDoc](https://godoc.org/github.com/lisitsky/go-site-search-string?status.svg)](https://godoc.org/github.com/lisitsky/go-site-search-string)


Using Gin framework <https://github.com/gin-gonic/gin> create a web server with a handler `/checkText`.
Handler will listen for `POST` request with such `JSON`:
```json
{
   "Site":["https://google.com","https://yahoo.com"],
   "SearchText":"Google"
}
```

Request structure:
```go
type Request struct {
    Site []string // Slice of strings: https://blog.golang.org/go-slices-usage-and-internals
    SearchText string
}
```

After getting request handler must get the body content of each website mentioned in `Site` variable (this is list of urls) and search in it for a `SearchText`. You can use default Go http client to get the websites body content.
* If the requested `SearchText` was found on the page of any `Site` url, webserver must return `JSON` with the url of the site at which text was found.

Response example:
```json
{
    "FoundAtSite":"https://google.com"
}
```

Response structure:
```go
type Response struct {
    FoundAtSite string
}
```

* If text was not found return `HTTP Code 204 No Content`.

Your test web-server must be provided at your Github repo. Just send as a link.

