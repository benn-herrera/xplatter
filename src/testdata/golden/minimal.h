#ifndef TEST_API_H
#define TEST_API_H

#include <stdint.h>
#include <stdbool.h>

typedef struct engine_s* engine_handle;

/* Platform services — implement these per platform */
void test_api_log_sink(int32_t level, const char* tag, const char* message);
uint32_t test_api_resource_count(void);
int32_t  test_api_resource_name(uint32_t index, char* buffer, uint32_t buffer_size);
int32_t  test_api_resource_exists(const char* name);
uint32_t test_api_resource_size(const char* name);
int32_t  test_api_resource_read(const char* name, uint8_t* buffer, uint32_t buffer_size);

/* lifecycle */
int32_t test_api_lifecycle_create_engine(engine_handle* out_result);
void test_api_lifecycle_destroy_engine(engine_handle engine);

#endif
