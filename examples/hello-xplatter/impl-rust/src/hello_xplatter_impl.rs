/*
 * Concrete implementation of the hello_xplatter traits.
 *
 * This hand-written file replaces the generated stub implementation.
 */

use std::ffi::{c_void, CString};
use crate::hello_xplatter_types::*;
use crate::hello_xplatter_trait::*;

/// Per-instance greeter state. The FFI shim (see generated/hello_xplatter_ffi.rs) boxes this
/// as the opaque handle: create_greeter → Box::new(Impl); destroy_greeter → Box::from_raw.
pub struct Impl {
    /// Holds the last formatted message. The pointer in HelloGreeting
    /// borrows from this CString, valid until the next say_hello call
    /// or destroy_greeter.
    message: Option<CString>,
}

impl Impl {
    pub fn new() -> Self {
        Impl { message: None }
    }
}

impl Greeter for Impl {
    fn say_hello(&self, greeter: *mut c_void, name: &str) -> Result<HelloGreeting, HelloErrorCode> {
        let state = unsafe { &mut *(greeter as *mut Impl) };

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
