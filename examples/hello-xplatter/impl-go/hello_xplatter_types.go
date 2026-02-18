package main

// Go enum constants matching the generated C header (hello_xplatter.h).
// The xplatter codegen tool generates these in a library package;
// this file adapts them for the example's package main.

const (
	HelloErrorCodeOk              = 0
	HelloErrorCodeInvalidArgument = 1
	HelloErrorCodeInternalError   = 2
)
