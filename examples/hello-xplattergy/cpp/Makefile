XPLATTERGY ?= ../../../bin/xplattergy
CXX        ?= c++
CC         ?= cc
CXXFLAGS   := -Wall -Wextra -std=c++20 -I. -Igenerated
CFLAGS     := -Wall -Wextra -std=c11 -I. -Igenerated

BUILD_DIR := build
STAMP     := $(BUILD_DIR)/.generated

# Hand-written C++ sources
CXX_SRCS := hello_xplattergy_impl.cpp main.cpp
# Generated C++ shim
GEN_CXX_SRCS := generated/hello_xplattergy_shim.cpp
# Platform services compiled as C
C_SRCS := platform_services.c

CXX_OBJS := $(addprefix $(BUILD_DIR)/,$(CXX_SRCS:.cpp=.o)) $(addprefix $(BUILD_DIR)/,$(notdir $(GEN_CXX_SRCS:.cpp=.o)))
C_OBJS   := $(addprefix $(BUILD_DIR)/,$(C_SRCS:.c=.o))
TARGET   := $(BUILD_DIR)/hello_xplattergy

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
  DYLIB_EXT := dylib
else
  DYLIB_EXT := so
endif
SHARED_LIB := $(BUILD_DIR)/libhello_xplattergy.$(DYLIB_EXT)

# Sources for shared library (everything except main.cpp)
LIB_CXX_SRCS := hello_xplattergy_impl.cpp
LIB_GEN_CXX_SRCS := generated/hello_xplattergy_shim.cpp
LIB_C_SRCS := platform_services.c
LIB_CXX_OBJS := $(addprefix $(BUILD_DIR)/lib_,$(notdir $(LIB_CXX_SRCS:.cpp=.o))) $(addprefix $(BUILD_DIR)/lib_,$(notdir $(LIB_GEN_CXX_SRCS:.cpp=.o)))
LIB_C_OBJS   := $(addprefix $(BUILD_DIR)/lib_,$(LIB_C_SRCS:.c=.o))

.PHONY: run shared-lib clean

run: $(TARGET)
	./$(TARGET)

shared-lib: $(SHARED_LIB)

$(SHARED_LIB): $(LIB_CXX_OBJS) $(LIB_C_OBJS)
	$(CXX) -shared -Wl,-install_name,@rpath/libhello_xplattergy.$(DYLIB_EXT) -o $@ $^

$(BUILD_DIR)/lib_%.o: %.cpp $(STAMP)
	@mkdir -p $(BUILD_DIR)
	$(CXX) $(CXXFLAGS) -fPIC -fvisibility=hidden -DHELLO_XPLATTERGY_BUILD -c -o $@ $<

$(BUILD_DIR)/lib_%.o: generated/%.cpp $(STAMP)
	@mkdir -p $(BUILD_DIR)
	$(CXX) $(CXXFLAGS) -fPIC -fvisibility=hidden -DHELLO_XPLATTERGY_BUILD -c -o $@ $<

$(BUILD_DIR)/lib_%.o: %.c $(STAMP)
	@mkdir -p $(BUILD_DIR)
	$(CC) $(CFLAGS) -fPIC -fvisibility=hidden -DHELLO_XPLATTERGY_BUILD -c -o $@ $<

$(TARGET): $(CXX_OBJS) $(C_OBJS)
	$(CXX) -o $@ $^

$(BUILD_DIR)/%.o: %.cpp $(STAMP)
	@mkdir -p $(BUILD_DIR)
	$(CXX) $(CXXFLAGS) -c -o $@ $<

$(BUILD_DIR)/%.o: generated/%.cpp $(STAMP)
	@mkdir -p $(BUILD_DIR)
	$(CXX) $(CXXFLAGS) -c -o $@ $<

$(BUILD_DIR)/%.o: %.c $(STAMP)
	@mkdir -p $(BUILD_DIR)
	$(CC) $(CFLAGS) -c -o $@ $<

$(STAMP): ../shared_schemas/hello_xplattergy.yaml
	@mkdir -p $(BUILD_DIR)
	$(XPLATTERGY) generate --impl-lang cpp --clean -o generated $<
	@touch $@

clean:
	rm -rf generated $(BUILD_DIR)
