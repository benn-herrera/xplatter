/*
 * Test driver for the C hello_xplatter example.
 *
 * Exercises the full lifecycle: create, greet, error case, destroy.
 */

#include "generated/hello_xplatter.h"

#include <stdio.h>
#include <string.h>

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

int main(void) {
    printf("=== hello_xplatter C example ===\n\n");

    /* Create a greeter */
    greeter_handle greeter = NULL;
    int32_t err = hello_xplatter_lifecycle_create_greeter(&greeter);
    CHECK(err == Hello_ErrorCode_Ok, "create_greeter succeeds");
    CHECK(greeter != NULL, "greeter handle is non-null");

    /* Say hello */
    Hello_Greeting greeting = {0};
    err = hello_xplatter_greeter_say_hello(greeter, "World", &greeting);
    CHECK(err == Hello_ErrorCode_Ok, "say_hello succeeds");
    CHECK(greeting.message != NULL, "greeting message is non-null");
    CHECK(strcmp(greeting.message, "Hello, World!") == 0, "greeting message is correct");

    /* Verify apiImpl */
    CHECK(greeting.apiImpl != NULL, "apiImpl is non-null");
    CHECK(strcmp(greeting.apiImpl, "impl-c") == 0, "apiImpl is correct");

    /* Say hello again (message buffer reused) */
    err = hello_xplatter_greeter_say_hello(greeter, "xplatter", &greeting);
    CHECK(err == Hello_ErrorCode_Ok, "say_hello succeeds again");
    CHECK(strcmp(greeting.message, "Hello, xplatter!") == 0, "greeting message updated");

    /* Empty name returns empty message (not error) */
    err = hello_xplatter_greeter_say_hello(greeter, "", &greeting);
    CHECK(err == Hello_ErrorCode_Ok, "empty name succeeds");
    CHECK(strcmp(greeting.message, "") == 0, "empty name gives empty message");
    CHECK(strcmp(greeting.apiImpl, "impl-c") == 0, "apiImpl set for empty name");

    /* Error case: null arguments */
    err = hello_xplatter_greeter_say_hello(NULL, "test", &greeting);
    CHECK(err == Hello_ErrorCode_InvalidArgument, "null greeter returns InvalidArgument");

    /* Destroy */
    hello_xplatter_lifecycle_destroy_greeter(greeter);
    printf("\n  Greeter destroyed.\n");

    printf("\n%d/%d tests passed.\n", tests_passed, tests_run);
    return tests_passed == tests_run ? 0 : 1;
}
