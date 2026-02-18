/*
 * Test driver for the C++ hello_xplattergy example.
 *
 * Calls through the C ABI (not the C++ interface directly) to exercise
 * the full shim → interface → impl path.
 */

#include "generated/hello_xplattergy.h"

#include <cstdio>
#include <cstring>

static int tests_run = 0;
static int tests_passed = 0;

#define CHECK(cond, msg) do { \
    tests_run++; \
    if (cond) { \
        tests_passed++; \
        printf("  PASS: %s\n", msg); \
    } else { \
        printf("  FAIL: %s\n", msg); \
    } \
} while(0)

int main() {
    printf("=== hello_xplattergy C++ example ===\n\n");

    /* Create a greeter (shim calls factory, returns handle) */
    greeter_handle greeter = nullptr;
    int32_t err = hello_xplattergy_lifecycle_create_greeter(&greeter);
    CHECK(err == Hello_ErrorCode_Ok, "create_greeter succeeds");
    CHECK(greeter != nullptr, "greeter handle is non-null");

    /* Say hello through C ABI → shim → interface → impl */
    Hello_Greeting greeting = {};
    err = hello_xplattergy_greeter_say_hello(greeter, "World", &greeting);
    CHECK(err == Hello_ErrorCode_Ok, "say_hello succeeds");
    CHECK(greeting.message != nullptr, "greeting message is non-null");
    CHECK(std::strcmp(greeting.message, "Hello, World!") == 0, "greeting message is correct");

    /* Verify apiImpl */
    CHECK(greeting.apiImpl != nullptr, "apiImpl is non-null");
    CHECK(std::strcmp(greeting.apiImpl, "impl-cpp") == 0, "apiImpl is correct");

    /* Say hello again */
    err = hello_xplattergy_greeter_say_hello(greeter, "xplattergy", &greeting);
    CHECK(err == Hello_ErrorCode_Ok, "say_hello succeeds again");
    CHECK(std::strcmp(greeting.message, "Hello, xplattergy!") == 0, "greeting message updated");

    /* Empty name returns empty message (not error) */
    err = hello_xplattergy_greeter_say_hello(greeter, "", &greeting);
    CHECK(err == Hello_ErrorCode_Ok, "empty name succeeds");
    CHECK(std::strcmp(greeting.message, "") == 0, "empty name gives empty message");
    CHECK(std::strcmp(greeting.apiImpl, "impl-cpp") == 0, "apiImpl set for empty name");

    /* Destroy (shim deletes the interface instance) */
    hello_xplattergy_lifecycle_destroy_greeter(greeter);
    printf("\n  Greeter destroyed.\n");

    printf("\n%d/%d tests passed.\n", tests_passed, tests_run);
    return tests_passed == tests_run ? 0 : 1;
}
