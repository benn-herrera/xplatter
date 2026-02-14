#ifndef TEST_API_H
#define TEST_API_H

#include <stdint.h>
#include <stdbool.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct engine_s* engine_handle;

typedef enum {
    Common_ErrorCode_Ok = 0,
    Common_ErrorCode_InvalidArgument = 1,
    Common_ErrorCode_OutOfMemory = 2,
    Common_ErrorCode_NotFound = 3,
    Common_ErrorCode_InternalError = 4
} Common_ErrorCode;

typedef enum {
    Common_LogLevel_Debug = 0,
    Common_LogLevel_Info = 1,
    Common_LogLevel_Warn = 2,
    Common_LogLevel_Error = 3
} Common_LogLevel;

typedef enum {
    Rendering_TextureFormat_RGBA8 = 0,
    Rendering_TextureFormat_RGB8 = 1,
    Rendering_TextureFormat_R8 = 2
} Rendering_TextureFormat;

typedef struct Geometry_Transform3D {
    float m00;
    float m01;
    float m02;
    float m03;
    float m10;
    float m11;
    float m12;
    float m13;
    float m20;
    float m21;
    float m22;
    float m23;
    float m30;
    float m31;
    float m32;
    float m33;
} Geometry_Transform3D;

typedef struct Common_EntityId {
    uint64_t id;
} Common_EntityId;

typedef struct Common_EventQueue {
    uint32_t capacity;
} Common_EventQueue;

typedef struct Input_TouchEvent {
    int32_t pointer_id;
    float x;
    float y;
    float pressure;
    uint64_t timestamp_ns;
} Input_TouchEvent;

typedef struct Input_TouchEventBatch {
    const TouchEvent* events;
    uint32_t events_count;
} Input_TouchEventBatch;

typedef struct Rendering_RendererConfig {
    uint32_t width;
    uint32_t height;
    bool vsync;
} Rendering_RendererConfig;

typedef struct Scene_EntityDefinition {
    const char* name;
} Scene_EntityDefinition;

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

#ifdef __cplusplus
}
#endif

#endif
