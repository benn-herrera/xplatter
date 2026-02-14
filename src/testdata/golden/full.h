#ifndef EXAMPLE_APP_ENGINE_H
#define EXAMPLE_APP_ENGINE_H

#include <stdint.h>
#include <stdbool.h>

typedef struct engine_s* engine_handle;
typedef struct renderer_s* renderer_handle;
typedef struct scene_s* scene_handle;
typedef struct texture_s* texture_handle;

/* Platform services — implement these per platform */
void example_app_engine_log_sink(int32_t level, const char* tag, const char* message);
uint32_t example_app_engine_resource_count(void);
int32_t  example_app_engine_resource_name(uint32_t index, char* buffer, uint32_t buffer_size);
int32_t  example_app_engine_resource_exists(const char* name);
uint32_t example_app_engine_resource_size(const char* name);
int32_t  example_app_engine_resource_read(const char* name, uint8_t* buffer, uint32_t buffer_size);

/* lifecycle */
int32_t example_app_engine_lifecycle_create_engine(engine_handle* out_result);
void example_app_engine_lifecycle_destroy_engine(engine_handle engine);

/* renderer */
int32_t example_app_engine_renderer_create_renderer(
    engine_handle engine,
    const Rendering_RendererConfig* config,
    renderer_handle* out_result);
void example_app_engine_renderer_destroy_renderer(renderer_handle renderer);
int32_t example_app_engine_renderer_begin_frame(renderer_handle renderer);
int32_t example_app_engine_renderer_end_frame(renderer_handle renderer);

/* texture */
int32_t example_app_engine_texture_load_texture_from_path(
    renderer_handle renderer,
    const char* path,
    texture_handle* out_result);
int32_t example_app_engine_texture_load_texture_from_buffer(
    renderer_handle renderer,
    const uint8_t* data,
    uint32_t data_len,
    Rendering_TextureFormat format,
    texture_handle* out_result);
void example_app_engine_texture_destroy_texture(texture_handle texture);

/* input */
int32_t example_app_engine_input_push_touch_events(
    engine_handle engine,
    const Input_TouchEventBatch* events);

/* events */
int32_t example_app_engine_events_poll_events(
    engine_handle engine,
    Common_EventQueue* events);

#endif
