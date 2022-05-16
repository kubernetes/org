/*
Copyright 2022 The Kubernetes Authors.

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
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5"
	"k8s.io/test-infra/prow/config/org"
	"sigs.k8s.io/yaml"
)

func stringInSlice(slice []string, key string) bool {
	for _, e := range slice {
		if key == e {
			return true
		}
	}

	return false
}

func stringInSliceCaseAgnostic(slice []string, key string) bool {
	for _, e := range slice {
		if strings.EqualFold(key, e) {
			return true
		}
	}

	return false
}

func findInvalidOrgs(orgs []string) []string {
	invalid := []string{}

	for _, org := range orgs {
		if !stringInSlice(validOrgs, org) {
			invalid = append(invalid, org)
		}
	}

	return invalid
}

func readConfig(path string) (*org.Config, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read file at %s: %s", path, err)
	}

	config := org.Config{}
	if err := yaml.Unmarshal(contents, &config); err != nil {
		return nil, fmt.Errorf("unable to unmarshal config from %s: %s", path, err)
	}

	return &config, nil
}

func saveConfig(path string, config *org.Config) error {
	b, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("unable to marshal config: %s", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("unable to fetch info for %s: %s", path, err)
	}

	if err := os.WriteFile(path, b, info.Mode()); err != nil {
		return fmt.Errorf("unable to write to %s: %s", path, err)
	}
	return nil
}

func commitChanges(repoRoot string, configsModified []string, message string) error {
	r, err := git.PlainOpen(repoRoot)
	if err != nil {
		return fmt.Errorf("unable to open repository: %s", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("unable to fetch worktree: %s", err)
	}

	for _, configModified := range configsModified {
		_, err := w.Add(configModified)
		if err != nil {
			return fmt.Errorf("unable to stage changes: %s", err)
		}
	}

	_, err = w.Commit(message, &git.CommitOptions{})
	if err != nil {
		return fmt.Errorf("unable to commit changes: %s", err)
	}

	return nil
}

func caseAgnosticSort(arr []string) {
	sort.Slice(arr, func(i, j int) bool {
		return strings.ToLower(arr[i]) < strings.ToLower(arr[j])
	})
}
