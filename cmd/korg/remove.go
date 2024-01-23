/*
Copyright 2024 The Kubernetes Authors.

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
	"path/filepath"
	"strings"
)

func RemoveMemberFromOrgs(o Options, username string) error {
	if invalidOrgs := findInvalidOrgs(o.Orgs); len(invalidOrgs) > 0 {
		return fmt.Errorf("specified invalid orgs: %s", strings.Join(invalidOrgs, ", "))
	}

	if !o.Confirm {
		fmt.Println("!!! running in dry-run mode. pass --confirm to persist changes.")
	}

	configsModified := []string{}
	for _, org := range o.Orgs {
		fmt.Printf("removing %s from %s org\n", username, org)

		relativeConfigPath := fmt.Sprintf(orgConfigPathFormat, org)
		configPath := filepath.Join(o.RepoRoot, relativeConfigPath)

		config, err := readConfig(configPath)
		if err != nil {
			return fmt.Errorf("reading config: %s", err)
		}

		if stringInSliceCaseAgnostic(config.Admins, username) {
			return fmt.Errorf("user %s is an admin for org %s", username, org)
		}

		if !stringInSliceCaseAgnostic(config.Members, username) {
			return fmt.Errorf("user %s doesn't exist in org %s", username, org)
		}

		// remove user from the org
		for i, member := range config.Members {
			if strings.EqualFold(username, member) {
				config.Members = append(config.Members[:i], config.Members[i+1:]...)
				break
			}
		}

		if o.Confirm {
			fmt.Printf("saving config for %s org\n", org)
			if err := saveConfig(configPath, config); err != nil {
				return fmt.Errorf("saving config: %s", err)
			}
		}

		configsModified = append(configsModified, relativeConfigPath)
		fmt.Printf("config files modified: %s\n", strings.Join(configsModified, ", "))
	}

	if o.Confirm {
		fmt.Println("committing changes")

		message := fmt.Sprintf("remove %s from %s", username, strings.Join(o.Orgs, ", "))
		if err := commitChanges(o.RepoRoot, configsModified, message); err != nil {
			return fmt.Errorf("committing changes: %s", err)
		}
	}
	return nil
}
