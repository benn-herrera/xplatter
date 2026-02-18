/*
 * C implementation of the hello_xplatter API.
 *
 * Directly implements the C ABI functions declared in the generated header.
 */

#include "generated/hello_xplatter.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

/* Internal greeter state */
typedef struct greeter_s {
    char message_buf[256];
} greeter_s;

int32_t hello_xplatter_lifecycle_create_greeter(greeter_handle* out_result) {
    greeter_s* g = (greeter_s*)calloc(1, sizeof(greeter_s));
    if (!g) {
        return Hello_ErrorCode_InternalError;
    }
    *out_result = g;
    return Hello_ErrorCode_Ok;
}

void hello_xplatter_lifecycle_destroy_greeter(greeter_handle greeter) {
    free(greeter);
}

int32_t hello_xplatter_greeter_say_hello(
    greeter_handle greeter,
    const char* name,
    Hello_Greeting* out_result)
{
    if (!greeter || !name || !out_result) {
        return Hello_ErrorCode_InvalidArgument;
    }

    if (name[0] == '\0') {
        out_result->message = "";
    } else {
        snprintf(greeter->message_buf, sizeof(greeter->message_buf),
                 "Hello, %s!", name);
        out_result->message = greeter->message_buf;
    }
    out_result->apiImpl = "impl-c";
    return Hello_ErrorCode_Ok;
}
