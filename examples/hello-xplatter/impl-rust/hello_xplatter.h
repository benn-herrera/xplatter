#ifndef HELLO_XPLATTER_H
#define HELLO_XPLATTER_H

#include <stdint.h>
#include <stdbool.h>

/* Symbol visibility */
#if defined(_WIN32) || defined(_WIN64)
  #ifdef HELLO_XPLATTER_BUILD
    #define HELLO_XPLATTER_EXPORT __declspec(dllexport)
  #else
    #define HELLO_XPLATTER_EXPORT __declspec(dllimport)
  #endif
#elif defined(__GNUC__) || defined(__clang__)
  #define HELLO_XPLATTER_EXPORT __attribute__((visibility("default")))
#else
  #define HELLO_XPLATTER_EXPORT
#endif

#ifdef __cplusplus
extern "C" {
#endif

typedef struct greeter_s* greeter_handle;

typedef enum {
    Hello_ErrorCode_Ok = 0,
    Hello_ErrorCode_InvalidArgument = 1,
    Hello_ErrorCode_InternalError = 2
} Hello_ErrorCode;

typedef struct Hello_Greeting {
    const char* message;
    const char* apiImpl;
} Hello_Greeting;

/* Platform services — implement these per platform */
void hello_xplatter_log_sink(int32_t level, const char* tag, const char* message);
uint32_t hello_xplatter_resource_count(void);
int32_t  hello_xplatter_resource_name(uint32_t index, char* buffer, uint32_t buffer_size);
int32_t  hello_xplatter_resource_exists(const char* name);
uint32_t hello_xplatter_resource_size(const char* name);
int32_t  hello_xplatter_resource_read(const char* name, uint8_t* buffer, uint32_t buffer_size);

/* lifecycle */
HELLO_XPLATTER_EXPORT int32_t hello_xplatter_lifecycle_create_greeter(
    greeter_handle* out_result);
HELLO_XPLATTER_EXPORT void hello_xplatter_lifecycle_destroy_greeter(
    greeter_handle greeter);

/* greeter */
HELLO_XPLATTER_EXPORT int32_t hello_xplatter_greeter_say_hello(
    greeter_handle greeter,
    const char* name,
    Hello_Greeting* out_result);

#ifdef __cplusplus
}
#endif

#endif
