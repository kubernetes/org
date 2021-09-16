# Copyright 2021 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

SHELL := /bin/bash

# available for override
GITHUB_TOKEN_PATH ?= /etc/github-token/token
TEST_INFRA_PATH ?= $(OUTPUT_DIR)/tmp/test-infra

# intentionally hardcoded list to ensure it's high friction to remove someone
ADMINS = cblecker fejta idvoretskyi mrbobbytables nikhita spiffxp
ORGS = $(shell find ./config -type d -mindepth 1 -maxdepth 1 | cut -d/ -f3)

# use absolute path to ./_output, which is .gitignored
OUTPUT_DIR := $(shell pwd)/_output
OUTPUT_BIN_DIR := $(OUTPUT_DIR)/bin

MERGE_CMD := $(OUTPUT_BIN_DIR)/merge
PERIBOLOS_CMD := $(OUTPUT_BIN_DIR)/peribolos

MERGED_CONFIG := $(OUTPUT_DIR)/gen-config.yaml

# convenience targets for humans
.PHONY: clean
clean:
	rm -rf $(OUTPUT_DIR)

.PHONY: build
build:
	go build ./...

.PHONY: merge
merge: $(MERGE_CMD)

.PHONY: config
config: $(MERGED_CONFIG)

.PHONY: peribolos
peribolos: $(PERIBOLOS_CMD)

.PHONY: test
test: config
	go test ./... --config=$(MERGED_CONFIG)

.PHONY: deploy # --confirm
deploy: config test peribolos
	$(PERIBOLOS_CMD) \
		--config-path $(MERGED_CONFIG) \
		--fix-org \
		--fix-org-members \
		--fix-teams \
		--fix-team-members \
		--github-token-path=$(GITHUB_TOKEN_PATH) \
		$(patsubst %, --required-admins=%, $(ADMINS)) \
		$@

# actual targets that only get built if they don't already exist
$(MERGE_CMD):
	mkdir -p "$(OUTPUT_BIN_DIR)"
	go build -v -o "$(OUTPUT_BIN_DIR)" ./cmd/merge

$(MERGED_CONFIG): $(MERGE_CMD) config/**/*.yaml
	mkdir -p "$(OUTPUT_DIR)"
	$(MERGE_CMD) \
		--merge-teams \
		$(shell for o in $(ORGS); do echo "--org-part=$$o=config/$$o/org.yaml"; done) \
		> $(MERGED_CONFIG)

$(TEST_INFRA_PATH):
	mkdir -p $(TEST_INFRA_PATH)
	git clone --depth=1 https://github.com/kubernetes/test-infra $(TEST_INFRA_PATH)

$(PERIBOLOS_CMD): $(TEST_INFRA_PATH)
	cd $(TEST_INFRA_PATH) && \
		go build -v -o $(PERIBOLOS_CMD) ./prow/cmd/peribolos
