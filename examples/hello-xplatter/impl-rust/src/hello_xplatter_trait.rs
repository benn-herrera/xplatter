/*
 * Trait definitions for the hello_xplatter API.
 *
 * Follows the pattern from the generated hello_xplatter_trait.rs,
 * with the necessary type imports added.
 */

use std::ffi::c_void;
use crate::hello_xplatter_types::*;

/// Lifecycle interface methods.
pub trait Lifecycle {
    fn create_greeter(&self) -> Result<*mut c_void, HelloErrorCode>;
    fn destroy_greeter(&self, greeter: *mut c_void);
}

/// Greeter interface methods.
pub trait Greeter {
    fn say_hello(&self, greeter: *mut c_void, name: &str) -> Result<HelloGreeting, HelloErrorCode>;
}
