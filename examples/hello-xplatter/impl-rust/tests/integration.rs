/*
 * Integration test for the hello_xplatter Rust example.
 *
 * Calls through the extern "C" FFI boundary to exercise the full
 * trait/FFI/impl stack.
 */

// Link the library crate to pull in the #[no_mangle] extern "C" symbols.
extern crate hello_xplatter;

use std::ffi::{c_void, CStr, CString};
use std::os::raw::c_char;

#[repr(C)]
struct HelloGreeting {
    message: *const c_char,
    #[allow(non_snake_case)]
    apiImpl: *const c_char,
}

extern "C" {
    fn hello_xplatter_lifecycle_create_greeter(out_result: *mut *mut c_void) -> i32;
    fn hello_xplatter_lifecycle_destroy_greeter(greeter: *mut c_void);
    fn hello_xplatter_greeter_say_hello(
        greeter: *mut c_void,
        name: *const c_char,
        out_result: *mut HelloGreeting,
    ) -> i32;
}

#[test]
fn test_create_and_destroy() {
    unsafe {
        let mut greeter: *mut c_void = std::ptr::null_mut();
        let err = hello_xplatter_lifecycle_create_greeter(&mut greeter);
        assert_eq!(err, 0, "create_greeter should succeed");
        assert!(!greeter.is_null(), "greeter should be non-null");

        hello_xplatter_lifecycle_destroy_greeter(greeter);
    }
}

#[test]
fn test_say_hello() {
    unsafe {
        let mut greeter: *mut c_void = std::ptr::null_mut();
        hello_xplatter_lifecycle_create_greeter(&mut greeter);

        let name = CString::new("World").unwrap();
        let mut greeting = HelloGreeting {
            message: std::ptr::null(),
            apiImpl: std::ptr::null(),
        };
        let err = hello_xplatter_greeter_say_hello(greeter, name.as_ptr(), &mut greeting);
        assert_eq!(err, 0, "say_hello should succeed");
        assert!(!greeting.message.is_null(), "message should be non-null");

        let msg = CStr::from_ptr(greeting.message).to_str().unwrap();
        assert_eq!(msg, "Hello, World!");

        assert!(!greeting.apiImpl.is_null(), "apiImpl should be non-null");
        let impl_name = CStr::from_ptr(greeting.apiImpl).to_str().unwrap();
        assert_eq!(impl_name, "impl-rust");

        hello_xplatter_lifecycle_destroy_greeter(greeter);
    }
}

#[test]
fn test_say_hello_twice() {
    unsafe {
        let mut greeter: *mut c_void = std::ptr::null_mut();
        hello_xplatter_lifecycle_create_greeter(&mut greeter);

        let name1 = CString::new("World").unwrap();
        let mut greeting = HelloGreeting {
            message: std::ptr::null(),
            apiImpl: std::ptr::null(),
        };
        hello_xplatter_greeter_say_hello(greeter, name1.as_ptr(), &mut greeting);

        let name2 = CString::new("xplatter").unwrap();
        hello_xplatter_greeter_say_hello(greeter, name2.as_ptr(), &mut greeting);
        let msg = CStr::from_ptr(greeting.message).to_str().unwrap();
        assert_eq!(msg, "Hello, xplatter!");

        hello_xplatter_lifecycle_destroy_greeter(greeter);
    }
}

#[test]
fn test_say_hello_empty_name() {
    unsafe {
        let mut greeter: *mut c_void = std::ptr::null_mut();
        hello_xplatter_lifecycle_create_greeter(&mut greeter);

        let name = CString::new("").unwrap();
        let mut greeting = HelloGreeting {
            message: std::ptr::null(),
            apiImpl: std::ptr::null(),
        };
        let err = hello_xplatter_greeter_say_hello(greeter, name.as_ptr(), &mut greeting);
        assert_eq!(err, 0, "empty name should succeed");
        assert!(!greeting.message.is_null(), "message should be non-null for empty name");
        let msg = CStr::from_ptr(greeting.message).to_str().unwrap();
        assert_eq!(msg, "", "empty name gives empty message");
        let impl_name = CStr::from_ptr(greeting.apiImpl).to_str().unwrap();
        assert_eq!(impl_name, "impl-rust", "apiImpl set for empty name");

        hello_xplatter_lifecycle_destroy_greeter(greeter);
    }
}
