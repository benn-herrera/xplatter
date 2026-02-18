/*
 * Android platform services for the hello_xplatter example.
 *
 * Logging uses __android_log_print; resource functions are stubs.
 */

#include <stdint.h>
#include <android/log.h>

void hello_xplatter_log_sink(int32_t level, const char* tag, const char* message) {
    int prio = (level <= 1) ? ANDROID_LOG_DEBUG : ANDROID_LOG_INFO;
    __android_log_print(prio, tag, "%s", message);
}

uint32_t hello_xplatter_resource_count(void) {
    return 0;
}

int32_t hello_xplatter_resource_name(uint32_t index, char* buffer, uint32_t buffer_size) {
    (void)index;
    (void)buffer;
    (void)buffer_size;
    return -1;
}

int32_t hello_xplatter_resource_exists(const char* name) {
    (void)name;
    return 0;
}

uint32_t hello_xplatter_resource_size(const char* name) {
    (void)name;
    return 0;
}

int32_t hello_xplatter_resource_read(const char* name, uint8_t* buffer, uint32_t buffer_size) {
    (void)name;
    (void)buffer;
    (void)buffer_size;
    return -1;
}
