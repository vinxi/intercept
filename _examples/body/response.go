package main

import (
	"fmt"
	"gopkg.in/vinxi/intercept.v0"
	"gopkg.in/vinxi/vinxi.v0"
	"strings"
)

func main() {
	fmt.Printf("Server listening on port: %d\n", 3100)
	vs := vinxi.NewServer(vinxi.ServerOptions{Host: "localhost", Port: 3100})

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
