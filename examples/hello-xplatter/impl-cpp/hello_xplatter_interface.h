#ifndef HELLO_XPLATTER_INTERFACE_H
#define HELLO_XPLATTER_INTERFACE_H

#include <stdint.h>
#include <stdbool.h>
#include <cstddef>
#include <string_view>
#include <span>
#include "hello_xplatter.h"

class HelloXplatterInterface {
public:
    virtual ~HelloXplatterInterface() = default;

    /* lifecycle */
    virtual int32_t create_greeter(void** out_result) = 0;
    virtual void destroy_greeter(void* greeter) = 0;

    /* greeter */
    virtual int32_t say_hello(void* greeter, std::string_view name, Hello_Greeting* out_result) = 0;

};

// Factory function — implement this to return your concrete instance.
HelloXplatterInterface* create_hello_xplatter_instance();

#endif
