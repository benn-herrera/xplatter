/*
 * Stub platform services for the hello_xplatter example.
 *
 * These are link-time functions declared in the generated C header.
 */

use std::os::raw::c_char;

#[no_mangle]
pub extern "C" fn hello_xplatter_log_sink(_level: i32, _tag: *const c_char, _message: *const c_char) {
}

#[no_mangle]
pub extern "C" fn hello_xplatter_resource_count() -> u32 {
    0
}

#[no_mangle]
pub extern "C" fn hello_xplatter_resource_name(_index: u32, _buffer: *mut c_char, _buffer_size: u32) -> i32 {
    -1
}

#[no_mangle]
pub extern "C" fn hello_xplatter_resource_exists(_name: *const c_char) -> i32 {
    0
}

#[no_mangle]
pub extern "C" fn hello_xplatter_resource_size(_name: *const c_char) -> u32 {
    0
}

#[no_mangle]
pub extern "C" fn hello_xplatter_resource_read(_name: *const c_char, _buffer: *mut u8, _buffer_size: u32) -> i32 {
    -1
}
