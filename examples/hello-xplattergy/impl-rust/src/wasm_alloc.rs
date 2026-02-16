// WASM malloc/free exports using a header-based allocator.
// Stores the allocation size in an 8-byte prefix so free() can reconstruct the Layout.

use std::alloc::{alloc, dealloc, Layout};

#[no_mangle]
pub unsafe extern "C" fn malloc(size: usize) -> *mut u8 {
    if size == 0 {
        return std::ptr::null_mut();
    }
    let total = size + 8;
    let layout = Layout::from_size_align_unchecked(total, 8);
    let ptr = alloc(layout);
    if ptr.is_null() {
        return std::ptr::null_mut();
    }
    *(ptr as *mut usize) = size;
    ptr.add(8)
}

#[no_mangle]
pub unsafe extern "C" fn free(ptr: *mut u8) {
    if ptr.is_null() {
        return;
    }
    let base = ptr.sub(8);
    let size = *(base as *mut usize);
    let total = size + 8;
    let layout = Layout::from_size_align_unchecked(total, 8);
    dealloc(base, layout);
}
