/*
Copyright 2018 The Kubernetes Authors.

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

package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/config/org"

	"github.com/ghodss/yaml"
)

type owners struct {
	Reviewers []string `json:"reviewers,omitempty"`
	Approvers []string `json:"approvers"`
}

func readInto(path string, i interface{}) error {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read: %v", err)
	}
	if err := yaml.Unmarshal(buf, i); err != nil {
		return fmt.Errorf("unmarshal: %v", err)
	}
	return nil
}

func loadOwners(dir string) (*owners, error) {
	var own owners
	if err := readInto(dir+"/OWNERS", &own); err != nil {
		return nil, err
	}
	return &own, nil
}

func loadOrg(dir string) (*org.Config, error) {
	var cfg org.Config
	if err := readInto(dir+"/org.yaml", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func testDuplicates(list []string) error {
	found := sets.String{}
	dups := sets.String{}
	for _, i := range list {
		if found.Has(i) {
			dups.Insert(i)
		}
		found.Insert(i)
	}
	if n := len(dups); n > 0 {
		return fmt.Errorf("%d duplicate names: %s", n, strings.Join(dups.List(), ", "))
	}
	return nil
}

func testOrg(targetDir string, t *testing.T) {
	cfg, err := loadOrg(targetDir)
	if err != nil {
		t.Fatalf("failed to load org.yaml: %v", err)
	}
	own, err := loadOwners(targetDir)
	if err != nil {
		t.Fatalf("failed to load OWNERS: %v", err)
	}

	members := sets.NewString(cfg.Members...)
	members.Insert(cfg.Admins...)

	reviewers := sets.NewString(own.Reviewers...)
	approvers := sets.NewString(own.Approvers...)

	if n := len(approvers); n < 5 {
		t.Errorf("Require at least 5 approvers, found %d: %s", n, strings.Join(approvers.List(), ", "))
	}

	if missing := reviewers.Difference(members); len(missing) > 0 {
		t.Errorf("The following reviewers must be members: %s", strings.Join(missing.List(), ", "))
	}
	if missing := approvers.Difference(members); len(missing) > 0 {
		t.Errorf("The following approvers must be members: %s", strings.Join(missing.List(), ", "))
	}

	admins := sets.NewString(cfg.Admins...)
	if !admins.Has("k8s-ci-robot") {
		t.Errorf("k8s-ci-robot must be an admin")
	}

	if cfg.BillingEmail != nil {
		t.Errorf("billing_email must be unset")
	}

	if err := testDuplicates(cfg.Admins); err != nil {
		t.Errorf("duplicate admins: %v", err)
	}
	if err := testDuplicates(cfg.Members); err != nil {
		t.Errorf("duplicate members: %v", err)
	}
	if err := testDuplicates(own.Reviewers); err != nil {
		t.Errorf("duplicate reviewers: %v", err)
	}
	if err := testDuplicates(own.Approvers); err != nil {
		t.Errorf("duplicate approvers: %v", err)
	}
}

func TestAllOrgs(t *testing.T) {
	cfg, err := config.Load("config.yaml", "")
	if err != nil {
		t.Fatalf("cannot read config.yaml from //config:gen-config.yaml: %v", err)
	}
	f, err := os.Open(".")
	if err != nil {
		t.Fatalf("cannot read config: %v", err)
	}
	infos, err := f.Readdir(0)
	if err != nil {
		t.Fatalf("cannot read subdirs: %v", err)
	}
	for _, i := range infos {
		if !i.IsDir() {
			continue
		}
		n := i.Name()
		if strings.HasPrefix(n, "linux_") || strings.HasPrefix(n, "darwin_") {
			continue
		}
		t.Run(n, func(t *testing.T) {
			if _, ok := cfg.Orgs[n]; !ok {
				t.Errorf("%s missing from generated config.yaml", n)
			}
			testOrg(n, t)
		})
	}
}
