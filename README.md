# intercept [![Build Status](https://travis-ci.org/vinci-proxy/intercept.png)](https://travis-ci.org/vinci-proxy/intercept) [![GitHub release](https://img.shields.io/badge/version-0.1.0-orange.svg?style=flat)](https://github.com/vinci-proxy/intercept/releases) [![GoDoc](https://godoc.org/github.com/vinci-proxy/intercept?status.svg)](https://godoc.org/github.com/vinci-proxy/intercept) [![Coverage Status](https://coveralls.io/repos/github/vinci-proxy/intercept/badge.svg?branch=master)](https://coveralls.io/github/vinci-proxy/intercept?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/vinci-proxy/intercept)](https://goreportcard.com/report/github.com/vinci-proxy/intercept)

Middleware to easily intercept and/or modify HTTP requests/responses before send them to the client/server.

## Installation

```bash
go get -u gopkg.in/vinci-proxy/intercept.v0
```

## API

See [godoc reference](https://godoc.org/github.com/vinci-proxy/intercept) for detailed API documentation.

## Examples

#### Response interceptor and modifier

```go
package main

import (
  "fmt"
  "gopkg.in/vinci-proxy/intercept.v0"
  "gopkg.in/vinci-proxy/vinci.v0"
  "strings"
)

func main() {
  fmt.Printf("Server listening on port: %d\n", 3100)
  vs := vinci.NewServer(vinci.ServerOptions{Host: "localhost", Port: 3100})

  // Intercept request and modify URI path
  vs.Use(intercept.Request(func(req *intercept.RequestModifier) {
    req.Request.RequestURI = "/html"
  }))

  // Intercept and replace response body
  vs.Use(intercept.Response(func(res *intercept.ResponseModifier) {
    data, _ := res.ReadString()
    str := strings.Replace(data, "Herman Melville - Moby-Dick", "A Long History", 1)
    res.String(str)
  }))

  vs.Forward("http://httpbin.org")

  err := vs.Listen()
  if err != nil {
    fmt.Errorf("Error: %s\n", err)
  }
}
```

## License

[MIT](https://opensource.org/licenses/MIT).
