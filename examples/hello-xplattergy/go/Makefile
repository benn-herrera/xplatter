XPLATTERGY ?= ../../../bin/xplattergy
BUILD_DIR  := build
STAMP      := $(BUILD_DIR)/.generated

.PHONY: run clean

run: $(STAMP)
	go build -o $(BUILD_DIR)/hello_xplattergy .
	./$(BUILD_DIR)/hello_xplattergy

# Generate and copy C header locally (cgo needs headers in the package directory)
$(STAMP): hello_xplattergy.yaml
	@mkdir -p $(BUILD_DIR)
	$(XPLATTERGY) generate --clean -o generated $<
	cp generated/hello_xplattergy.h .
	@touch $@

clean:
	rm -rf generated $(BUILD_DIR) hello_xplattergy.h
