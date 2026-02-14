XPLATTERGY ?= ../../../bin/xplattergy
CC         ?= cc
CFLAGS     := -Wall -Wextra -std=c11 -I. -Igenerated

SRCS      := greeter_impl.c platform_services.c main.c
BUILD_DIR := build
STAMP     := $(BUILD_DIR)/.generated
TARGET    := $(BUILD_DIR)/hello_xplattergy

.PHONY: run clean

run: $(TARGET)
	./$(TARGET)

$(TARGET): $(SRCS) $(STAMP)
	@mkdir -p $(BUILD_DIR)
	$(CC) $(CFLAGS) -o $@ $(SRCS)

$(STAMP): hello_xplattergy.yaml
	@mkdir -p $(BUILD_DIR)
	$(XPLATTERGY) generate --clean -o generated $<
	@touch $@

clean:
	rm -rf generated $(BUILD_DIR)
