cmake_minimum_required(VERSION 3.15)
project(hello-xplatter VERSION 0.1.0 LANGUAGES C)

set(CMAKE_C_STANDARD 17)
set(CMAKE_C_STANDARD_REQUIRED ON)

if(EMSCRIPTEN)
    add_executable(hello_xplatter
        hello_xplatter_impl.c
        platform_services/web.c
    )
    set_target_properties(hello_xplatter PROPERTIES SUFFIX ".wasm")
    target_compile_options(hello_xplatter PRIVATE
        -std=c17 -Wall -Wextra -fvisibility=hidden
    )
    target_link_options(hello_xplatter PRIVATE
        "SHELL:--no-entry"
        "SHELL:-s EXPORTED_FUNCTIONS=_malloc,_free,_hello_xplatter_lifecycle_create_greeter,_hello_xplatter_lifecycle_destroy_greeter,_hello_xplatter_greeter_say_hello"
        "SHELL:-s STANDALONE_WASM"
        "SHELL:-O2"
    )
    target_include_directories(hello_xplatter PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}
        ${CMAKE_CURRENT_SOURCE_DIR}/generated
    )
endif()
