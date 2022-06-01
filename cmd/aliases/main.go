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

// FROM https://github.com/knative/community/blob/main/mechanics/tools/gen-aliases/main.go

package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
	peribolos "k8s.io/test-infra/prow/config/org"
	"sigs.k8s.io/yaml"
)

func main() {
	if len(os.Args) < 2 {
		os.Args = append(os.Args, "kubernetes")
	}
	org := os.Args[1]
	if len(os.Args) < 3 {
		os.Args = append(os.Args, filepath.Join("config", os.Args[1]+"/org.yaml"))
	}
	if len(os.Args) < 4 {
		os.Args = append(os.Args, "OWNERS_ALIASES")
	}
	infile, outfile := os.Args[2], os.Args[3]

	f, err := ioutil.ReadFile(infile)
	if err != nil {
		log.Print("Unable to open ", err)
		os.Exit(1)
	}
	var cfg peribolos.FullConfig
	if err := yaml.Unmarshal(f, &cfg); err != nil {
		log.Print("Unable to parse ", err)
		os.Exit(1)
	}
	out := AliasConfig{
		Aliases: map[string][]string{},
	}
	dashes := func(r rune) rune {
		if r >= 'a' && r <= 'z' {
			return r
		}
		if r >= '0' && r <= '9' {
			return r
		}
		return '-'
	}
	for _, team := range expandTeams(cfg.Orgs[org].Teams) {
		name := strings.Map(dashes, strings.ToLower(team.Name))
		out.Aliases[name] = team.Members.List()
	}

	output, err := yaml.Marshal(out)
	if err != nil {
		log.Print("Could not serialize: ", err)
		os.Exit(1)
	}
	preamble := `# This file is auto-generated from peribolos.
# Do not modify this file, instead modify the peribolos config in k/org repository.

`
	output = append([]byte(preamble), output...)
	ioutil.WriteFile(outfile, output, 0644)
	log.Print("Wrote ", outfile)
}

func expandTeams(seed map[string]peribolos.Team) []expandedTeam {
	var retval []expandedTeam
	for name, team := range seed {
		children := expandTeams(team.Children)
		this := expandedTeam{
			Name:    name,
			Members: sets.NewString(team.Members...),
		}
		this.Members.Insert(team.Maintainers...)
		for _, child := range children {
			this.Members = this.Members.Union(child.Members)
		}
		retval = append(retval, this)
		retval = append(retval, children...)
	}

	return retval
}

type AliasConfig struct {
	Aliases map[string][]string `json:"aliases"`
}

type expandedTeam struct {
	Name    string
	Members sets.String
}
