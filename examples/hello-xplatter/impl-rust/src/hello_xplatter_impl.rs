/*
 * Concrete implementation of the hello_xplatter traits.
 *
 * This hand-written file replaces the generated stub implementation.
 */

use std::ffi::{c_void, CString};
use crate::hello_xplatter_types::*;
use crate::hello_xplatter_trait::*;

/// ZST dispatch target — all trait calls go through this.
pub struct Impl;

/// Internal state for a greeter instance.
struct GreeterState {
    /// Holds the last formatted message. The pointer in HelloGreeting
    /// borrows from this CString, valid until the next say_hello call
    /// or destroy_greeter.
    message: Option<CString>,
}

impl Lifecycle for Impl {
    fn create_greeter(&self) -> Result<*mut c_void, HelloErrorCode> {
        let state = Box::new(GreeterState { message: None });
        Ok(Box::into_raw(state) as *mut c_void)
    }

    fn destroy_greeter(&self, greeter: *mut c_void) {
        unsafe {
            drop(Box::from_raw(greeter as *mut GreeterState));
        }
    }
}

impl Greeter for Impl {
    fn say_hello(&self, greeter: *mut c_void, name: &str) -> Result<HelloGreeting, HelloErrorCode> {
        let state = unsafe { &mut *(greeter as *mut GreeterState) };

        if name.is_empty() {
            state.message = Some(CString::new("").unwrap());
        } else {
            let msg = CString::new(format!("Hello, {}!", name))
                .map_err(|_| HelloErrorCode::InternalError)?;
            state.message = Some(msg);
        }
        let ptr = state.message.as_ref().unwrap().as_ptr();

        Ok(HelloGreeting { message: ptr, apiImpl: b"impl-rust\0".as_ptr() as *const _ })
    }
}
