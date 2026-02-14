/*
 * Concrete implementation of the HelloXplattergyInterface.
 *
 * This hand-written file overrides the generated stub (via -I. -Igenerated/).
 */

#ifndef HELLO_XPLATTERGY_IMPL_H
#define HELLO_XPLATTERGY_IMPL_H

#include "generated/hello_xplattergy.h"
#include "hello_xplattergy_interface.h"

#include <string>

class HelloXplattergyImpl : public HelloXplattergyInterface {
public:
    HelloXplattergyImpl();
    ~HelloXplattergyImpl() override;

    /* lifecycle — not called by the shim (handled directly) */
    int32_t create_greeter(void** out_result) override;
    void destroy_greeter(void* greeter) override;

    /* greeter */
    int32_t say_hello(void* greeter, std::string_view name, Hello_Greeting* out_result) override;

private:
    std::string message_buf_;
};

#endif
