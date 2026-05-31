package main

import (
	"context"
	"os"

	"github.com/w0rxbend/ubuntu-bootstrap/internal/app"
)

func main() {
	os.Exit(app.Run(context.Background(), os.Args[1:], app.Streams{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}))
}
