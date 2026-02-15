/*
 * Web/WASM platform services for the hello_xplattergy example.
 *
 * No-op stubs compiled into the WASM binary. Since these are resolved
 * at compile time, the JS _buildPlatformImports() entries are unused.
 */

#include <stdint.h>

void hello_xplattergy_log_sink(int32_t level, const char* tag, const char* message) {
    (void)level; (void)tag; (void)message;
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
