/*
 * C++ desktop terminal app that loads the hello_xplattergy shared library
 * and exercises the API as a consumer would — via the C ABI.
 */

#include "hello_xplattergy.h"

#include <cstdio>
#include <cstring>

int main() {
    std::printf("=== hello_xplattergy desktop app (C++) ===\n\n");

    // Create a greeter handle via the C ABI
    greeter_handle greeter = nullptr;
    int32_t err = hello_xplattergy_lifecycle_create_greeter(&greeter);
    if (err != Hello_ErrorCode_Ok || !greeter) {
        std::fprintf(stderr, "Failed to create greeter (error %d)\n", err);
        return 1;
    }

    // Discover backing implementation
    Hello_Greeting probe = {};
    err = hello_xplattergy_greeter_say_hello(greeter, "", &probe);
    if (err == Hello_ErrorCode_Ok && probe.apiImpl) {
        std::printf("Backing implementation: %s\n", probe.apiImpl);
    }

    char buf[256];
    std::printf("Enter a name (or 'exit' to quit): ");
    std::fflush(stdout);

    while (std::fgets(buf, sizeof(buf), stdin)) {
        // Strip trailing newline
        size_t len = std::strlen(buf);
        if (len > 0 && buf[len - 1] == '\n') {
            buf[len - 1] = '\0';
            len--;
        }

        if (std::strcmp(buf, "exit") == 0 || std::strcmp(buf, "quit") == 0) {
            break;
        }

        if (len == 0) {
            std::printf("Enter a name (or 'exit' to quit): ");
            std::fflush(stdout);
            continue;
        }

        Hello_Greeting result = {};
        err = hello_xplattergy_greeter_say_hello(greeter, buf, &result);
        if (err != Hello_ErrorCode_Ok) {
            std::fprintf(stderr, "say_hello failed (error %d)\n", err);
        } else {
            std::printf("%s\n", result.message);
        }

        std::printf("Enter a name (or 'exit' to quit): ");
        std::fflush(stdout);
    }

    hello_xplattergy_lifecycle_destroy_greeter(greeter);
    std::printf("Goodbye!\n");
    return 0;
}
