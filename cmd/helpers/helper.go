/*
Copyright 2021 The Kubernetes Authors.

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

package helpers

import (
	"fmt"
	"io/ioutil"
	"strings"

	"k8s.io/test-infra/prow/config/org"

	"github.com/ghodss/yaml"
)

func ParseKeyValue(s string) (string, string) {
	p := strings.SplitN(s, "=", 2)
	if len(p) == 1 {
		return p[0], ""
	}
	return p[0], p[1]
}

type FlagMap map[string]string

func (fm FlagMap) String() string {
	var parts []string
	for key, value := range fm {
		if value == "" {
			parts = append(parts, key)
			continue
		}
		parts = append(parts, key+"="+value)
	}
	return strings.Join(parts, ",")
}

func (fm FlagMap) Set(s string) error {
	k, v := ParseKeyValue(s)
	if _, present := fm[k]; present {
		return fmt.Errorf("duplicate key: %s", k)
	}
	fm[k] = v
	return nil
}

func UnmarshalPathToOrgConfig(path string) (*org.Config, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %v", err)
	}
	var cfg org.Config
	if err := yaml.Unmarshal(buf, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal: %v", err)
	}
	return &cfg, nil
}
