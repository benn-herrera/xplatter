/*
 * Concrete implementation of the HelloXplatterInterface.
 *
 * This hand-written file overrides the generated stub (via -I. -Igenerated/).
 */

#include "hello_xplatter_impl.h"

HelloXplatterImpl::HelloXplatterImpl() = default;
HelloXplatterImpl::~HelloXplatterImpl() = default;

/* These lifecycle methods exist on the interface but are never called
   by the generated shim — create/destroy are handled directly. */
int32_t HelloXplatterImpl::create_greeter(void** /*out_result*/) {
    return Hello_ErrorCode_Ok;
}

void HelloXplatterImpl::destroy_greeter(void* /*greeter*/) {
}

int32_t HelloXplatterImpl::say_hello(
    void* /*greeter*/,
    std::string_view name,
    Hello_Greeting* out_result)
{
    if (!out_result) {
        return Hello_ErrorCode_InvalidArgument;
    }

    if (name.empty()) {
        message_buf_.clear();
    } else {
        message_buf_ = "Hello, ";
        message_buf_ += name;
        message_buf_ += "!";
    }

    out_result->message = message_buf_.c_str();
    out_result->apiImpl = "impl-cpp";
    return Hello_ErrorCode_Ok;
}

/* Factory function — returns a new instance of the implementation. */
HelloXplatterInterface* create_hello_xplatter_instance() {
    return new HelloXplatterImpl();
}
