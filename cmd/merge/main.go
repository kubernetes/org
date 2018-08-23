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

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"strings"

	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/config/org"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
)

func parseKeyValue(s string) (string, string) {
	p := strings.SplitN(s, "=", 2)
	if len(p) == 1 {
		return p[0], ""
	}
	return p[0], p[1]
}

type flagMap map[string]string

func (fm flagMap) String() string {
	var parts []string
	for key, value := range fm {
		if value == "" {
			parts = append(parts, key)
		}
		parts = append(parts, key+"="+value)
	}
	return strings.Join(parts, ",")
}

func (fm flagMap) Set(s string) error {
	k, v := parseKeyValue(s)
	if _, present := fm[k]; present {
		return fmt.Errorf("duplicate key: %s", k)
	}
	fm[k] = v
	return nil
}

func main() {
	orgs := flagMap{}
	flag.Var(orgs, "org-part", "Each instance adds an org-name=org.yaml part")
	flag.Parse()

	for _, a := range flag.Args() {
		logrus.Print("Extra", a)
		orgs.Set(a)
	}

	cfg, err := loadOrgs(orgs)
	if err != nil {
		logrus.Fatalf("Failed to load orgs: %v", err)
	}
	pc := config.ProwConfig{
		Orgs: cfg,
	}
	out, err := yaml.Marshal(pc)
	fmt.Println(string(out))
}

func loadOrgs(parts map[string]string) (map[string]org.Config, error) {
	config := map[string]org.Config{}
	for name, path := range parts {
		buf, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %v", path, err)
		}
		var cfg org.Config
		if err := yaml.Unmarshal(buf, &cfg); err != nil {
			return nil, fmt.Errorf("unmarshal %s: %v", path, err)
		}
		cfg.Teams = nil // TODO(fejta): support reading team parts
		config[name] = cfg
	}
	return config, nil
}
