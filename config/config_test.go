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
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/config/org"
	"k8s.io/test-infra/prow/github"

	"github.com/ghodss/yaml"
)

var configPath = flag.String("config", "config.yaml", "Path to generated config")

var cfg org.FullConfig

func TestMain(m *testing.M) {
	flag.Parse()
	if *configPath == "" {
		fmt.Println("--config must be set")
		os.Exit(1)
	}

	raw, err := ioutil.ReadFile(*configPath)
	if err != nil {
		fmt.Printf("cannot read generated config.yaml from %s: %v\n", *configPath, err)
		os.Exit(1)
	}

	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		fmt.Printf("cannot unmarshal generated config.yaml from %s: %v\n", *configPath, err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

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

func testDuplicates(list sets.String) error {
	found := sets.String{}
	dups := sets.String{}
	all := list.List()
	for _, i := range all {
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

func isSorted(list []string) bool {
	items := make([]string, len(list))
	for _, l := range list {
		items = append(items, strings.ToLower(l))
	}

	return sort.StringsAreSorted(items)
}

func normalize(s sets.String) sets.String {
	out := sets.String{}
	for i := range s {
		out.Insert(github.NormLogin(i))
	}
	return out
}

// testTeamMembers ensures that a user is not a maintainer and member at the same time,
// there are no duplicate names in the list and all users are org members.
func testTeamMembers(teams map[string]org.Team, admins sets.String, orgMembers sets.String, orgName string) []error {
	var errs []error
	for teamName, team := range teams {
		teamMaintainers := sets.NewString(team.Maintainers...)
		teamMembers := sets.NewString(team.Members...)

		teamMaintainers = normalize(teamMaintainers)
		teamMembers = normalize(teamMembers)

		// ensure all teams have privacy as closed
		if team.Privacy == nil || (team.Privacy != nil && *team.Privacy != org.Closed) {
			errs = append(errs, fmt.Errorf("The team %s in org %s doesn't have the `privacy: closed` field", teamName, orgName))
		}

		// check for non-admins in maintainers list
		if nonAdminMaintainers := teamMaintainers.Difference(admins); len(nonAdminMaintainers) > 0 {
			errs = append(errs, fmt.Errorf("The team %s in org %s has non-admins listed as maintainers; these users should be in the members list instead: %s", teamName, orgName, strings.Join(nonAdminMaintainers.List(), ",")))
		}

		// check for users in both maintainers and members
		if both := teamMaintainers.Intersection(teamMembers); len(both) > 0 {
			errs = append(errs, fmt.Errorf("The team %s in org %s has users in both maintainer admin and member roles: %s", teamName, orgName, strings.Join(both.List(), ", ")))
		}

		// check for duplicates
		if err := testDuplicates(teamMaintainers); err != nil {
			errs = append(errs, fmt.Errorf("The team %s in org %s has duplicate maintainers: %v", teamName, orgName, err))
		}
		if err := testDuplicates(teamMembers); err != nil {
			errs = append(errs, fmt.Errorf("The team %s in org %s has duplicate members: %v", teamMembers, orgName, err))
		}

		// check if all are org members
		if missing := teamMembers.Difference(orgMembers); len(missing) > 0 {
			errs = append(errs, fmt.Errorf("The following members of team %s are not %s org members: %s", teamName, orgName, strings.Join(missing.List(), ", ")))
		}

		// check if admins are a regular member of team
		if adminTeamMembers := teamMembers.Intersection(admins); len(adminTeamMembers) > 0 {
			errs = append(errs, fmt.Errorf("The team %s in org %s has org admins listed as members; these users should be in the maintainers list instead, and cannot be on the members list: %s", teamName, orgName, strings.Join(adminTeamMembers.List(), ", ")))
		}

		// check if lists are sorted
		if !isSorted(team.Maintainers) {
			errs = append(errs, fmt.Errorf("The team %s in org %s has an unsorted list of maintainers", teamName, orgName))
		}
		if !isSorted(team.Members) {
			errs = append(errs, fmt.Errorf("The team %s in org %s has an unsorted list of members", teamName, orgName))
		}

		if team.Children != nil {
			errs = append(errs, testTeamMembers(team.Children, admins, orgMembers, orgName)...)
		}
	}
	return errs
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

	members := normalize(sets.NewString(cfg.Members...))
	admins := normalize(sets.NewString(cfg.Admins...))
	allOrgMembers := members.Union(admins)

	reviewers := normalize(sets.NewString(own.Reviewers...))
	approvers := normalize(sets.NewString(own.Approvers...))

	if n := len(approvers); n < 5 {
		t.Errorf("Require at least 5 approvers, found %d: %s", n, strings.Join(approvers.List(), ", "))
	}

	if missing := reviewers.Difference(allOrgMembers); len(missing) > 0 {
		t.Errorf("The following reviewers must be members: %s", strings.Join(missing.List(), ", "))
	}
	if missing := approvers.Difference(allOrgMembers); len(missing) > 0 {
		t.Errorf("The following approvers must be members: %s", strings.Join(missing.List(), ", "))
	}
	if err := testDuplicates(reviewers); err != nil {
		t.Errorf("duplicate reviewers: %v", err)
	}
	if err := testDuplicates(approvers); err != nil {
		t.Errorf("duplicate approvers: %v", err)
	}
}

func TestAllOrgs(t *testing.T) {
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

	for _, org := range cfg.Orgs {
		members := normalize(sets.NewString(org.Members...))
		admins := normalize(sets.NewString(org.Admins...))
		allOrgMembers := members.Union(admins)

		if both := admins.Intersection(members); len(both) > 0 {
			t.Errorf("users in both org admin and member roles for org '%s': %s", *org.Name, strings.Join(both.List(), ", "))
		}

		if !admins.Has("k8s-ci-robot") {
			t.Errorf("k8s-ci-robot must be an admin")
		}

		if err := testDuplicates(admins); err != nil {
			t.Errorf("duplicate admins: %v", err)
		}
		if err := testDuplicates(allOrgMembers); err != nil {
			t.Errorf("duplicate members: %v", err)
		}
		if !isSorted(org.Admins) {
			t.Errorf("admins for %s org are unsorted", *org.Name)
		}
		if !isSorted(org.Members) {
			t.Errorf("members for %s org are unsorted", *org.Name)
		}

		if errs := testTeamMembers(org.Teams, admins, allOrgMembers, *org.Name); errs != nil {
			for _, err := range errs {
				t.Error(err)
			}
		}

	}
}
