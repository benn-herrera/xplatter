#ifndef EXAMPLE_APP_ENGINE_H
#define EXAMPLE_APP_ENGINE_H

#include <stdint.h>
#include <stdbool.h>

/* Symbol visibility */
#if defined(_WIN32) || defined(_WIN64)
  #ifdef EXAMPLE_APP_ENGINE_BUILD
    #define EXAMPLE_APP_ENGINE_EXPORT __declspec(dllexport)
  #else
    #define EXAMPLE_APP_ENGINE_EXPORT __declspec(dllimport)
  #endif
#elif defined(__GNUC__) || defined(__clang__)
  #define EXAMPLE_APP_ENGINE_EXPORT __attribute__((visibility("default")))
#else
  #define EXAMPLE_APP_ENGINE_EXPORT
#endif

#ifdef __cplusplus
extern "C" {
#endif

typedef struct engine_s* engine_handle;
typedef struct renderer_s* renderer_handle;
typedef struct scene_s* scene_handle;
typedef struct texture_s* texture_handle;

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
void example_app_engine_log_sink(int32_t level, const char* tag, const char* message);
uint32_t example_app_engine_resource_count(void);
int32_t  example_app_engine_resource_name(uint32_t index, char* buffer, uint32_t buffer_size);
int32_t  example_app_engine_resource_exists(const char* name);
uint32_t example_app_engine_resource_size(const char* name);
int32_t  example_app_engine_resource_read(const char* name, uint8_t* buffer, uint32_t buffer_size);

/* lifecycle */
EXAMPLE_APP_ENGINE_EXPORT int32_t example_app_engine_lifecycle_create_engine(
    engine_handle* out_result);
EXAMPLE_APP_ENGINE_EXPORT void example_app_engine_lifecycle_destroy_engine(
    engine_handle engine);

/* renderer */
EXAMPLE_APP_ENGINE_EXPORT int32_t example_app_engine_renderer_create_renderer(
    engine_handle engine,
    const Rendering_RendererConfig* config,
    renderer_handle* out_result);
EXAMPLE_APP_ENGINE_EXPORT void example_app_engine_renderer_destroy_renderer(
    renderer_handle renderer);
EXAMPLE_APP_ENGINE_EXPORT int32_t example_app_engine_renderer_begin_frame(
    renderer_handle renderer);
EXAMPLE_APP_ENGINE_EXPORT int32_t example_app_engine_renderer_end_frame(
    renderer_handle renderer);

/* texture */
EXAMPLE_APP_ENGINE_EXPORT int32_t example_app_engine_texture_load_texture_from_path(
    renderer_handle renderer,
    const char* path,
    texture_handle* out_result);
EXAMPLE_APP_ENGINE_EXPORT int32_t example_app_engine_texture_load_texture_from_buffer(
    renderer_handle renderer,
    const uint8_t* data,
    uint32_t data_len,
    Rendering_TextureFormat format,
    texture_handle* out_result);
EXAMPLE_APP_ENGINE_EXPORT void example_app_engine_texture_destroy_texture(
    texture_handle texture);

/* input */
EXAMPLE_APP_ENGINE_EXPORT int32_t example_app_engine_input_push_touch_events(
    engine_handle engine,
    const Input_TouchEventBatch* events);

/* events */
EXAMPLE_APP_ENGINE_EXPORT int32_t example_app_engine_events_poll_events(
    engine_handle engine,
    Common_EventQueue* events);

#ifdef __cplusplus
}
#endif

#endif
