/*
 * Concrete implementation of the HelloXplattergyInterface.
 *
 * This hand-written file overrides the generated stub (via -I. -Igenerated/).
 */

#include "hello_xplattergy_impl.h"

HelloXplattergyImpl::HelloXplattergyImpl() = default;
HelloXplattergyImpl::~HelloXplattergyImpl() = default;

/* These lifecycle methods exist on the interface but are never called
   by the generated shim — create/destroy are handled directly. */
int32_t HelloXplattergyImpl::create_greeter(void** /*out_result*/) {
    return Hello_ErrorCode_Ok;
}

void HelloXplattergyImpl::destroy_greeter(void* /*greeter*/) {
}

int32_t HelloXplattergyImpl::say_hello(
    void* /*greeter*/,
    std::string_view name,
    Hello_Greeting* out_result)
{
    if (name.empty() || !out_result) {
        return Hello_ErrorCode_InvalidArgument;
    }

    message_buf_ = "Hello from impl-cpp, ";
    message_buf_ += name;
    message_buf_ += "!";

    out_result->message = message_buf_.c_str();
    return Hello_ErrorCode_Ok;
}

/* Factory function — returns a new instance of the implementation. */
HelloXplattergyInterface* create_hello_xplattergy_instance() {
    return new HelloXplattergyImpl();
}
