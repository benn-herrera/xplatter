package main

/*
#include <stdint.h>
*/
import "C"

//export hello_xplatter_log_sink
func hello_xplatter_log_sink(level C.int32_t, tag *C.char, message *C.char) {
}

//export hello_xplatter_resource_count
func hello_xplatter_resource_count() C.uint32_t {
	return 0
}

//export hello_xplatter_resource_name
func hello_xplatter_resource_name(index C.uint32_t, buffer *C.char, bufferSize C.uint32_t) C.int32_t {
	return -1
}

//export hello_xplatter_resource_exists
func hello_xplatter_resource_exists(name *C.char) C.int32_t {
	return 0
}

//export hello_xplatter_resource_size
func hello_xplatter_resource_size(name *C.char) C.uint32_t {
	return 0
}

//export hello_xplatter_resource_read
func hello_xplatter_resource_read(name *C.char, buffer *C.uint8_t, bufferSize C.uint32_t) C.int32_t {
	return -1
}
