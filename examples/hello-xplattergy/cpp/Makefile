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

.PHONY: run clean

run: $(TARGET)
	./$(TARGET)

$(TARGET): $(CXX_OBJS) $(C_OBJS)
	$(CXX) -o $@ $^

$(BUILD_DIR)/%.o: %.cpp $(STAMP)
	$(CXX) $(CXXFLAGS) -c -o $@ $<

$(BUILD_DIR)/%.o: generated/%.cpp $(STAMP)
	$(CXX) $(CXXFLAGS) -c -o $@ $<

$(BUILD_DIR)/%.o: %.c $(STAMP)
	$(CC) $(CFLAGS) -c -o $@ $<

$(STAMP): hello_xplattergy.yaml
	@mkdir -p $(BUILD_DIR)
	$(XPLATTERGY) generate --clean -o generated $<
	@touch $@

clean:
	rm -rf generated $(BUILD_DIR)
