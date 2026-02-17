//go:build wasip1
package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"
)

// GreeterWASMImpl holds the state for a greeter instance in the WASM build.
type GreeterWASMImpl struct {
	lastMsgPtr uintptr // WASM linear memory pointer to last allocated message (0 if none)
}

// Memory allocation: pin byte slices so the GC does not collect them.
var _wasmAllocs sync.Map

//go:wasmexport malloc
func _wasmMalloc(size uint32) uintptr {
	buf := make([]byte, size)
	ptr := uintptr(unsafe.Pointer(&buf[0]))
	_wasmAllocs.Store(ptr, buf)
	return ptr
}

//go:wasmexport free
func _wasmFree(ptr uintptr) {
	_wasmAllocs.Delete(ptr)
}

// Handle management: maps integer keys to Go objects.
// Integer keys avoid storing Go pointers in C/WASM memory.
var (
	_wasmHandles sync.Map
	_nextHandle  atomic.Uintptr
)

func _allocHandle(impl any) uintptr {
	key := _nextHandle.Add(1)
	_wasmHandles.Store(key, impl)
	return key
}

func _lookupHandle(key uintptr) (any, bool) {
	return _wasmHandles.Load(key)
}

func _freeHandle(key uintptr) {
	_wasmHandles.Delete(key)
}

// _cstring reads a null-terminated C string from WASM linear memory.
func _cstring(ptr uintptr) string {
	var n int
	for *(*byte)(unsafe.Pointer(ptr + uintptr(n))) != 0 {
		n++
	}
	return string(unsafe.Slice((*byte)(unsafe.Pointer(ptr)), n))
}

// Platform service imports — provided by the JS binding as WASM imports.

//go:wasmimport env hello_xplattergy_log_sink
func _platformLogSink(level int32, tag uintptr, message uintptr)

//go:wasmimport env hello_xplattergy_resource_count
func _platformResourceCount() uint32

//go:wasmimport env hello_xplattergy_resource_name
func _platformResourceName(index uint32, buffer uintptr, bufferSize uint32) int32

//go:wasmimport env hello_xplattergy_resource_exists
func _platformResourceExists(name uintptr) int32

//go:wasmimport env hello_xplattergy_resource_size
func _platformResourceSize(name uintptr) uint32

//go:wasmimport env hello_xplattergy_resource_read
func _platformResourceRead(name uintptr, buffer uintptr, bufferSize uint32) int32

/* lifecycle */

//go:wasmexport hello_xplattergy_lifecycle_create_greeter
func hello_xplattergy_lifecycle_create_greeter(out_result uintptr) int32 {
	impl := &GreeterWASMImpl{}
	handle := _allocHandle(impl)
	// Write handle into greeter_handle output slot (uint32 in WASM32)
	*(*uint32)(unsafe.Pointer(out_result)) = uint32(handle)
	return 0 // HelloErrorCodeOk
}

//go:wasmexport hello_xplattergy_lifecycle_destroy_greeter
func hello_xplattergy_lifecycle_destroy_greeter(greeter uintptr) {
	if val, ok := _lookupHandle(greeter); ok {
		impl := val.(*GreeterWASMImpl)
		if impl.lastMsgPtr != 0 {
			_wasmFree(impl.lastMsgPtr)
		}
	}
	_freeHandle(greeter)
}

/* greeter */

//go:wasmexport hello_xplattergy_greeter_say_hello
func hello_xplattergy_greeter_say_hello(greeter uintptr, name uintptr, out_result uintptr) int32 {
	val, ok := _lookupHandle(greeter)
	if !ok {
		return 1 // HelloErrorCodeInvalidArgument
	}
	impl := val.(*GreeterWASMImpl)

	goName := _cstring(name)
	if goName == "" {
		return 1 // HelloErrorCodeInvalidArgument
	}

	// Free previous message before allocating a new one
	if impl.lastMsgPtr != 0 {
		_wasmFree(impl.lastMsgPtr)
	}

	// Allocate and populate the message string in WASM linear memory
	msg := fmt.Sprintf("Hello from impl-go-wasm, %s!", goName)
	msgBytes := []byte(msg)
	msgLen := uint32(len(msgBytes))
	msgPtr := _wasmMalloc(msgLen + 1) // +1 for null terminator
	buf := unsafe.Slice((*byte)(unsafe.Pointer(msgPtr)), msgLen+1)
	copy(buf, msgBytes)
	buf[msgLen] = 0
	impl.lastMsgPtr = msgPtr

	// Write Hello_Greeting{message: msgPtr} to out_result.
	// In WASM32, Hello_Greeting is {uint32 message} at offset 0.
	*(*uint32)(unsafe.Pointer(out_result)) = uint32(msgPtr)
	return 0 // HelloErrorCodeOk
}

// main is the required entry point for a wasip1 WASM binary.
// The actual API surface is exported via //go:wasmexport directives above.
func main() {}
