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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/prow/pkg/config/org"
	"sigs.k8s.io/yaml"

	"github.com/hound-search/hound/client"
)

func stringInSlice(slice []string, key string) bool {
	for _, e := range slice {
		if key == e {
			return true
		}
	}

	return false
}

// Note for the future: once we bump to the latest go version, we can replace this with helpers from stdlib slice package
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

func IsOwner(username string) (bool, error) {
	url := fmt.Sprintf("https://cs.k8s.io/api/v1/search?stats=fosho&repos=*&rng=:20&q=%s&i=fosho&files=OWNERS&excludeFiles=vendor/", username)
	resp, err := http.Get(url)
	if err != nil {
		return false, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var r client.Response

	err = json.Unmarshal(body, &r)
	if err != nil {
		return false, err
	}

	return r.Stats.FilesOpened > 0, nil
}

func unmarshalFromFile(path string) (*org.Config, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %v", err)
	}

	return unmarshal(buf)
}

func unmarshal(buf []byte) (*org.Config, error) {
	var cfg org.Config
	if err := yaml.Unmarshal(buf, &cfg, yaml.DisallowUnknownFields); err != nil {
		return nil, fmt.Errorf("unmarshal: %v", err)
	}
	return &cfg, nil
}

func LoadOrgs(o Options) (map[string]org.Config, error) {
	config := map[string]org.Config{}
	for _, orgName := range o.Orgs {
		path := fmt.Sprintf("%s/config/%s/org.yaml", o.RepoRoot, orgName)

		cfg, err := unmarshalFromFile(path)
		if err != nil {
			return nil, fmt.Errorf("error in %s: %v", path, err)
		}

		if cfg.Teams == nil {
			cfg.Teams = map[string]org.Team{}
		}
		prefix := filepath.Dir(path)
		err = filepath.Walk(prefix, func(path string, info os.FileInfo, err error) error {
			switch {
			case path == prefix:
				return nil // Skip base dir
			case info.IsDir() && filepath.Dir(path) != prefix:
				logrus.Infof("Skipping %s and its children", path)
				return filepath.SkipDir // Skip prefix/foo/bar/ dirs
			case !info.IsDir() && filepath.Dir(path) == prefix:
				return nil // Ignore prefix/foo files
			case filepath.Base(path) == "teams.yaml":
				teamCfg, err := unmarshalFromFile(path)
				if err != nil {
					return fmt.Errorf("error in %s: %v", path, err)
				}

				for name, team := range teamCfg.Teams {
					cfg.Teams[name] = team
				}
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("merge teams %s: %v", path, err)
		}

		config[orgName] = *cfg
	}
	return config, nil
}
