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

SHELL := /usr/bin/env bash

# available for override
GITHUB_TOKEN_PATH ?=

# intentionally hardcoded list to ensure it's high friction to remove someone
ADMINS = cblecker jasonbraganza MadhavJivrajani mrbobbytables nikhita palnabarun Priyankasaggu11929
ORGS = $(shell find ./config -type d -mindepth 1 -maxdepth 1 | cut -d/ -f3)

# use absolute path to ./_output, which is .gitignored
OUTPUT_DIR := $(shell pwd)/_output
OUTPUT_BIN_DIR := $(OUTPUT_DIR)/bin

MERGE_CMD := $(OUTPUT_BIN_DIR)/merge
PERIBOLOS_CMD := $(OUTPUT_BIN_DIR)/peribolos

CONFIG_FILES = $(shell find config/ -type f -name '*.yaml')
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
	MERGED_CONFIG=$(MERGED_CONFIG) go test ./...

.PHONY: verify
verify:
	./hack/verify.sh

.PHONY: update-prep
update-prep: config test peribolos

.PHONY: deploy # --confirm
deploy:
	./admin/update.sh
		$(-*-command-variables-*-) $(filter-out $@,$(MAKECMDGOALS))

add-members:
	./hack/add-members.sh

# actual targets that only get built if they don't already exist
$(MERGE_CMD):
	mkdir -p "$(OUTPUT_BIN_DIR)"
	go build -o "$(OUTPUT_BIN_DIR)" ./cmd/merge

$(MERGED_CONFIG): clean $(MERGE_CMD) $(CONFIG_FILES)
	mkdir -p "$(OUTPUT_DIR)"
	$(MERGE_CMD) \
		--merge-teams \
		$(shell for o in $(ORGS); do echo "--org-part=$$o=config/$$o/org.yaml"; done) \
		> $(MERGED_CONFIG)

$(PERIBOLOS_CMD):
	GOBIN=$(OUTPUT_BIN_DIR) go install sigs.k8s.io/prow/cmd/peribolos@main
