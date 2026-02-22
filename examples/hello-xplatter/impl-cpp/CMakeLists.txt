cmake_minimum_required(VERSION 3.15)
project(hello-xplatter VERSION 0.1.0 LANGUAGES C CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

if(EMSCRIPTEN)
    add_executable(hello_xplatter
        hello_xplatter_impl.cpp
        generated/hello_xplatter_shim.cpp
        platform_services/web.c
    )
    set_target_properties(hello_xplatter PROPERTIES SUFFIX ".wasm")
    target_compile_options(hello_xplatter PRIVATE
        $<$<COMPILE_LANGUAGE:CXX>:-std=c++20 -Wall -Wextra -fvisibility=hidden>
        $<$<COMPILE_LANGUAGE:C>:-std=c17 -Wall -Wextra -fvisibility=hidden>
    )
    target_link_options(hello_xplatter PRIVATE
        "SHELL:--no-entry"
        "SHELL:-s EXPORTED_FUNCTIONS=_malloc,_free,_hello_xplatter_lifecycle_create_greeter,_hello_xplatter_lifecycle_destroy_greeter,_hello_xplatter_greeter_say_hello"
        "SHELL:-s STANDALONE_WASM"
        "SHELL:-O2"
    )
else()
    add_library(hello_xplatter SHARED
        generated/hello_xplatter_shim.cpp
        hello_xplatter_impl.cpp
    )
endif()

target_include_directories(hello_xplatter PRIVATE ${CMAKE_CURRENT_SOURCE_DIR} ${CMAKE_CURRENT_SOURCE_DIR}/generated)
