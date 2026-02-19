package main

/*
#include <stdlib.h>
#include "generated/hello_xplatter.h"
*/
import "C"
import (
	"fmt"
	"os"
	"unsafe"
)

func main() {
	fmt.Println("=== hello_xplatter Go example ===")
	fmt.Println()

	testsRun := 0
	testsPassed := 0

	check := func(cond bool, msg string) {
		testsRun++
		if cond {
			testsPassed++
			fmt.Printf("  PASS: %s\n", msg)
		} else {
			fmt.Printf("  FAIL: %s\n", msg)
		}
	}

	// Create a greeter (calls Go //export function directly)
	var greeter C.greeter_handle
	err := hello_xplatter_lifecycle_create_greeter(&greeter)
	check(err == C.int32_t(HelloErrorCodeOk), "create_greeter succeeds")
	check(greeter != nil, "greeter handle is non-nil")

	// Say hello
	name := C.CString("World")
	defer C.free(unsafe.Pointer(name))
	var greeting C.Hello_Greeting
	err = hello_xplatter_greeter_say_hello(greeter, name, &greeting)
	check(err == C.int32_t(HelloErrorCodeOk), "say_hello succeeds")
	check(greeting.message != nil, "greeting message is non-nil")
	msg := C.GoString(greeting.message)
	check(msg == "Hello, World!", "greeting message is correct")

	// Verify apiImpl
	check(greeting.apiImpl != nil, "apiImpl is non-nil")
	implName := C.GoString(greeting.apiImpl)
	check(implName == "impl-go", "apiImpl is correct")

	// Say hello again
	name2 := C.CString("xplatter")
	defer C.free(unsafe.Pointer(name2))
	err = hello_xplatter_greeter_say_hello(greeter, name2, &greeting)
	check(err == C.int32_t(HelloErrorCodeOk), "say_hello succeeds again")
	msg = C.GoString(greeting.message)
	check(msg == "Hello, xplatter!", "greeting message updated")

	// Empty name returns empty message (not error)
	emptyName := C.CString("")
	defer C.free(unsafe.Pointer(emptyName))
	err = hello_xplatter_greeter_say_hello(greeter, emptyName, &greeting)
	check(err == C.int32_t(HelloErrorCodeOk), "empty name succeeds")
	msg = C.GoString(greeting.message)
	check(msg == "", "empty name gives empty message")
	implName = C.GoString(greeting.apiImpl)
	check(implName == "impl-go", "apiImpl set for empty name")

	// Destroy
	hello_xplatter_lifecycle_destroy_greeter(greeter)
	fmt.Println()
	fmt.Println("  Greeter destroyed.")

	fmt.Printf("\n%d/%d tests passed.\n", testsPassed, testsRun)
	if testsPassed != testsRun {
		os.Exit(1)
	}
}
