WAILS ?= wails
GOCACHE ?= /tmp/go-build-cache
WAILS_TAGS ?= webkit2_41
APP_NAME ?= counting-embroidery-threads

TAG_FLAGS := $(if $(strip $(WAILS_TAGS)),-tags "$(WAILS_TAGS)",)

.PHONY: dev build-linux build-windows

dev:
	GOCACHE=$(GOCACHE) $(WAILS) dev $(TAG_FLAGS)

build-linux:
	GOCACHE=$(GOCACHE) $(WAILS) build -platform linux/amd64 $(TAG_FLAGS) -o $(APP_NAME)

build-windows:
	GOCACHE=$(GOCACHE) $(WAILS) build -platform windows/amd64 $(TAG_FLAGS) -o $(APP_NAME).exe
