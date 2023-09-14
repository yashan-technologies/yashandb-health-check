# env defines
GOOS=$(shell go env GOOS)
ARCH=$(shell arch)
VERSION=$(shell cat ./VERSION)
GO_VERSION=$(shell go env GOVERSION)
GIT_COMMIT_ID=$(shell git rev-parse HEAD)
GIT_DESCRIBE=$(shell git describe --always)
OS=$(if $(GOOS),$(GOOS),linux)

# go command defines
GO_BUILD=go build
GO_MOD_TIDY=$(go mod tidy -compat 1.19)
GO_BUILD_WITH_INFO=$(GO_BUILD) -ldflags "\
	-X 'yhc/defs/compiledef._appVersion=$(VERSION)' \
	-X 'yhc/defs/compiledef._goVersion=$(GO_VERSION)'\
	-X 'yhc/defs/compiledef._gitCommitID=$(GIT_COMMIT_ID)'\
	-X 'yhc/defs/compiledef._gitDescribe=$(GIT_DESCRIBE)'"

# package defines
PKG_PERFIX=yashan-health-check
PKG=$(PKG_PERFIX)-$(VERSION)-$(OS)-$(ARCH).tar.gz

BUILD_PATH=./build
PKG_PATH=$(BUILD_PATH)/$(PKG_PERFIX)
BIN_PATH=$(PKG_PATH)/bin
LOG_PATH=$(PKG_PATH)/log
DOCS_PATH=$(PKG_PATH)/docs
RESULTS_PATH=$(PKG_PATH)/results

# build defines
BIN_YHCCTL=$(BUILD_PATH)/yhcctl
BIN_FILES=$(BIN_YHCCTL)

DIR_TO_MAKE=$(BIN_PATH) $(LOG_PATH) $(RESULTS_PATH) $(DOCS_PATH)
FILE_TO_COPY=./config ./scripts ./static

.PHONY: clean force go_build

# functions
build: go_build
	@mkdir -p $(DIR_TO_MAKE) 
	@cp -r $(FILE_TO_COPY) $(PKG_PATH)
	# @cp -r ./yhc-doc $(DOCS_PATH)/markdown
	# @cp ./yhc.pdf $(DOCS_PATH)
	@mv $(BIN_FILES) $(BIN_PATH)
	@> $(LOG_PATH)/yhcctl.log
	@cd $(PKG_PATH);ln -s ./bin/yhcctl ./yhcctl
	@cd $(BUILD_PATH);tar -cvzf $(PKG) $(PKG_PERFIX)/

clean:
	rm -rf $(BUILD_PATH)

go_build: 
	$(GO_MOD_TIDY)
	$(GO_BUILD_WITH_INFO) -o $(BIN_YHCCTL) ./cmd/yhcctl/*.go

force: clean build