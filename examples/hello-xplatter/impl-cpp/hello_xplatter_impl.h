/*
 * Concrete implementation of the HelloXplatterInterface.
 *
 * This hand-written file overrides the generated stub (via -I. -Igenerated/).
 */

#ifndef HELLO_XPLATTER_IMPL_H
#define HELLO_XPLATTER_IMPL_H

#include "generated/hello_xplatter.h"
#include "hello_xplatter_interface.h"

#include <string>

class HelloXplatterImpl : public HelloXplatterInterface {
public:
    HelloXplatterImpl();
    ~HelloXplatterImpl() override;

    /* lifecycle — not called by the shim (handled directly) */
    int32_t create_greeter(void** out_result) override;
    void destroy_greeter(void* greeter) override;

    /* greeter */
    int32_t say_hello(void* greeter, std::string_view name, Hello_Greeting* out_result) override;

private:
    std::string message_buf_;
};

#endif
