cmake_minimum_required(VERSION 3.15)
project(hello-xplatter VERSION 0.1.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

add_library(hello-xplatter SHARED
    hello_xplatter_shim.cpp
    hello_xplatter_impl.cpp
)

target_include_directories(hello-xplatter PRIVATE ${CMAKE_CURRENT_SOURCE_DIR})
