
DIST=$(realpath ./dist)
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0

MODULE := $(shell go mod edit -print | grep ^module | awk '{print $$2}')
RELEASE_VER? := dev
BUILD_TAG := $(shell git describe --all --long | cut -d "-" -f 3)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# RPM=0

.PHONY: all cmd

#all: test
#	@echo ALL

cmd:
	@echo building $(path)...

ifeq ("$(wildcard $(path))","")
	$(error path to $(path) does not exist)
endif

	$(eval name := $(word $(words $(subst /, ,$(path))),$(subst /, ,$(path))))
	$(eval fullname := $(name)-$(RELEASE_VER)-$(GOOS)-$(GOARCH))

	@echo $(path) :: $(name) :: $(DIST)/$(path) :: $(fullname)

	mkdir -p $(DIST)/$(path)
	cd $(path); go build -ldflags "-X main.tag=$(BUILD_TAG) -X main.date=$(BUILD_DATE) -X main.version=$(RELEASE_VER)" -o $(DIST)/$(path)/$(fullname) .
	cd $(DIST)/$(path); sha512sum $(fullname) > $(fullname).sha512; md5sum $(fullname) > $(fullname).md5
	
