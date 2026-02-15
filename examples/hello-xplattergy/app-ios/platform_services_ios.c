/*
 * iOS platform services for the hello_xplattergy example.
 *
 * Logging uses os_log; resource functions are stubs.
 */

#include <stdint.h>
#include <os/log.h>

void hello_xplattergy_log_sink(int32_t level, const char* tag, const char* message) {
    os_log_type_t type = (level <= 1) ? OS_LOG_TYPE_DEBUG : OS_LOG_TYPE_DEFAULT;
    os_log_with_type(OS_LOG_DEFAULT, type, "[%{public}s] %{public}s", tag, message);
}

uint32_t hello_xplattergy_resource_count(void) {
    return 0;
}

int32_t hello_xplattergy_resource_name(uint32_t index, char* buffer, uint32_t buffer_size) {
    (void)index;
    (void)buffer;
    (void)buffer_size;
    return -1;
}

int32_t hello_xplattergy_resource_exists(const char* name) {
    (void)name;
    return 0;
}

uint32_t hello_xplattergy_resource_size(const char* name) {
    (void)name;
    return 0;
}

int32_t hello_xplattergy_resource_read(const char* name, uint8_t* buffer, uint32_t buffer_size) {
    (void)name;
    (void)buffer;
    (void)buffer_size;
    return -1;
}
