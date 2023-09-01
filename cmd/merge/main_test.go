/*
Copyright 2023 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"fmt"
	"testing"
)

const testOrgConfig = `admins:
- admin1
- admin2
billing_email: github@kubernetes.io
default_repository_permission: read
description: Org desc
has_organization_projects: true
has_repository_projects: true
members:
- member1
- member2
members_can_create_repositories: false
name: Org
teams:
  team-abc:
    description: team-abc desc
    members:
    - team-member1
    privacy: closed
    %s:
      abc: write
`

func TestStrictUnmarshalling(t *testing.T) {
	cases := []struct {
		repoKey     string
		expectError bool
		desc        string
	}{
		{
			repoKey:     "repos",
			expectError: false,
			desc:        "with a valid field",
		},
		{
			repoKey:     "somethingBizzare",
			expectError: true,
			desc:        "with an invalid field",
		},
	}

	for _, c := range cases {
		_, err := unmarshal(
			bytes.NewBufferString(fmt.Sprintf(testOrgConfig, c.repoKey)).Bytes(),
		)
		if !c.expectError && err != nil {
			t.Errorf("unexpected error for %s: %v", c.desc, err)
		}
		if c.expectError && err == nil {
			t.Errorf("expected error for %s", c.desc)
		}
	}
}
