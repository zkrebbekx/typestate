// Package main shows a typestate builder generated from an annotated struct.
//
// Run `go generate ./...` (with the typestate generator installed, or via the
// Makefile target) to regenerate server_builder.go after changing this struct.
package main

import (
	"fmt"
	"time"
)

//go:generate typestate -type Server

// Server is configured through a generated, compile-time-safe builder.
// Addr and Handler are required; the rest are optional.
type Server struct {
	Addr    string        `typestate:"required"`
	Handler string        `typestate:"required"`
	Timeout time.Duration // optional
	MaxConn int           // optional
}

func main() {
	// BuildServer only compiles because both required fields are set.
	s := BuildServer(NewServerBuilder().
		WithAddr(":8080").
		WithHandler("mux").
		WithTimeout(5 * time.Second))

	fmt.Printf("%+v\n", s)
}
