package main

/*
#include <stdlib.h>

// Local typedefs matching the generated C header (hello_xplattergy.h).
// We can't include that header directly because cgo's generated
// export prototypes would conflict with the header's declarations.
typedef struct greeter_s* greeter_handle;

typedef enum {
    Hello_ErrorCode_Ok = 0,
    Hello_ErrorCode_InvalidArgument = 1,
    Hello_ErrorCode_InternalError = 2
} Hello_ErrorCode;

typedef struct Hello_Greeting {
    const char* message;
} Hello_Greeting;
*/
import "C"
import (
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"
)

// GreeterImpl holds the state for a greeter instance.
type GreeterImpl struct {
	lastMessage *C.char // C-allocated string for borrowing semantics
}

// Handle management: maps integer keys to Go objects.
// We use integer keys (not real pointers) because Go pointers
// must not be stored in C memory.
var (
	handles    sync.Map
	nextHandle atomic.Uintptr
)

func allocHandle(impl interface{}) C.greeter_handle {
	key := nextHandle.Add(1)
	handles.Store(key, impl)
	return C.greeter_handle(unsafe.Pointer(key))
}

func lookupHandle(h C.greeter_handle) (*GreeterImpl, bool) {
	val, ok := handles.Load(uintptr(unsafe.Pointer(h)))
	if !ok {
		return nil, false
	}
	impl, ok := val.(*GreeterImpl)
	return impl, ok
}

func freeHandle(h C.greeter_handle) {
	key := uintptr(unsafe.Pointer(h))
	if val, ok := handles.LoadAndDelete(key); ok {
		if g, ok := val.(*GreeterImpl); ok {
			if g.lastMessage != nil {
				C.free(unsafe.Pointer(g.lastMessage))
			}
		}
	}
}

//export hello_xplattergy_lifecycle_create_greeter
func hello_xplattergy_lifecycle_create_greeter(out_result *C.greeter_handle) C.int32_t {
	impl := &GreeterImpl{}
	*out_result = allocHandle(impl)
	return C.int32_t(HelloErrorCodeOk)
}

//export hello_xplattergy_lifecycle_destroy_greeter
func hello_xplattergy_lifecycle_destroy_greeter(greeter C.greeter_handle) {
	freeHandle(greeter)
}

//export hello_xplattergy_greeter_say_hello
func hello_xplattergy_greeter_say_hello(greeter C.greeter_handle, name *C.char, out_result *C.Hello_Greeting) C.int32_t {
	impl, ok := lookupHandle(greeter)
	if !ok {
		return C.int32_t(HelloErrorCodeInvalidArgument)
	}

	goName := C.GoString(name)
	if goName == "" {
		return C.int32_t(HelloErrorCodeInvalidArgument)
	}

	// Free previous message
	if impl.lastMessage != nil {
		C.free(unsafe.Pointer(impl.lastMessage))
	}

	// Format message and store as C string (caller borrows)
	msg := fmt.Sprintf("Hello from impl-go, %s!", goName)
	impl.lastMessage = C.CString(msg)
	out_result.message = impl.lastMessage

	return C.int32_t(HelloErrorCodeOk)
}
