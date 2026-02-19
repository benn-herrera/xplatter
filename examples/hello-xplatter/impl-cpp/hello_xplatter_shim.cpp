#include "hello_xplatter_interface.h"
#include "hello_xplatter.h"

extern "C" {

/* lifecycle */
HELLO_XPLATTER_EXPORT int32_t hello_xplatter_lifecycle_create_greeter(greeter_handle* out_result) {
    HelloXplatterInterface* instance = create_hello_xplatter_instance();
    if (!instance) {
        return -1;
    }
    *out_result = reinterpret_cast<greeter_handle>(instance);
    return 0;
}

HELLO_XPLATTER_EXPORT void hello_xplatter_lifecycle_destroy_greeter(greeter_handle greeter) {
    HelloXplatterInterface* instance = reinterpret_cast<HelloXplatterInterface*>(greeter);
    delete instance;
}

/* greeter */
HELLO_XPLATTER_EXPORT int32_t hello_xplatter_greeter_say_hello(greeter_handle greeter, const char* name, Hello_Greeting* out_result) {
    HelloXplatterInterface* self = reinterpret_cast<HelloXplatterInterface*>(greeter);
    return self->say_hello(greeter, std::string_view(name), out_result);
}

} // extern "C"
