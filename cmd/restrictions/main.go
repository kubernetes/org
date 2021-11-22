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

package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"k8s.io/org/cmd/helpers"

	"github.com/bmatcuk/doublestar"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
)

var (
	emptyRegexp        = regexp.MustCompile("")
	defaultRestriction = Restriction{Path: "*", AllowedReposRe: []*regexp.Regexp{emptyRegexp}}
)

type Config struct {
	Restrictions []Restriction `json:"restrictions"`
}

type Restriction struct {
	Path           string   `json:"path"`
	AllowedRepos   []string `json:"allowedRepos,omitempty"`
	AllowedReposRe []*regexp.Regexp
}

type options struct {
	orgs         helpers.FlagMap
	restrictions string
}

func main() {
	o := options{orgs: helpers.FlagMap{}}
	flag.Var(o.orgs, "orgs", "Each instance adds an org-name=org.yaml part")
	flag.StringVar(&o.restrictions, "restrictions", "restrictions.yaml", "path to a configuration file containing restrictions")
	flag.Parse()

	for _, a := range flag.Args() {
		o.orgs.Set(a)
	}

	cfg, err := unmarshalPathToRestrictionsConfig(o.restrictions)
	if err != nil {
		logrus.Fatalf("Failed to unmarshal restrictions config: %v", err)
	}

	restrictions, err := compileRegexps(cfg.Restrictions)
	if err != nil {
		logrus.Fatalf("Failed to compile regexp for restrictions config: %v", err)
	}

	for name, path := range o.orgs {
		logrus.Infof("Validating restrictions for %s org", name)
		prefix := filepath.Dir(path)
		err := filepath.Walk(prefix, func(path string, info os.FileInfo, err error) error {
			switch {
			case path == prefix:
				return nil // Skip base dir
			case info.IsDir() && filepath.Dir(path) != prefix:
				logrus.Infof("Skipping %s and its children", path)
				return filepath.SkipDir // Skip prefix/foo/bar/ dirs
			case !info.IsDir() && filepath.Dir(path) == prefix && filepath.Base(path) != "org.yaml":
				return nil // Ignore prefix/foo files
			case filepath.Base(path) == "teams.yaml" || filepath.Base(path) == "org.yaml":
				if err := resolveRestriction(restrictions, path); err != nil {
					logrus.Error(err)
				}
			}
			return nil
		})
		if err != nil {
			logrus.Fatalf("Failed to walk through files at %s", path)
		}
	}

}

func unmarshalPathToRestrictionsConfig(path string) (*Config, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read restrictions config: %v", err)
	}
	var restrictionsCfg Config
	if err := yaml.Unmarshal(buf, &restrictionsCfg); err != nil {
		return nil, fmt.Errorf("unmarshal restrictions config: %v", err)
	}
	return &restrictionsCfg, nil
}

func compileRegexps(restrictions []Restriction) ([]Restriction, error) {
	ret := make([]Restriction, 0, len(restrictions))
	for _, r := range restrictions {
		r.AllowedReposRe = make([]*regexp.Regexp, 0, len(r.AllowedRepos))
		for _, repo := range r.AllowedRepos {
			re, err := regexp.Compile(repo)
			if err != nil {
				return restrictions, fmt.Errorf("failed to parse repo pattern %q: %v", repo, err)
			}
			r.AllowedReposRe = append(r.AllowedReposRe, re)
		}
		ret = append(ret, r)
	}
	return ret, nil
}

func resolveRestriction(restrictions []Restriction, path string) error {
	orgCfg, err := helpers.UnmarshalPathToOrgConfig(path)
	if err != nil {
		return fmt.Errorf("error in unmarshalling path %s: %v", path, err)
	}
	r := getRestrictionForPath(restrictions, path)

	var err2 error
	for teamName, team := range orgCfg.Teams {
		for repo := range team.Repos {
			if !matchesRegexList(repo, r.AllowedReposRe) {
				if err2 == nil {
					err2 = errors.New("") // needed to ensure %w is not nil below
				}
				err2 = fmt.Errorf("%w\n%q: cannot define repo %q for team %q", err2, path, repo, teamName)
			}
		}
	}
	return err2
}

func getRestrictionForPath(restrictions []Restriction, path string) Restriction {
	for _, r := range restrictions {
		if match, err := doublestar.Match(r.Path, path); err == nil && match {
			return r
		}
	}
	return defaultRestriction
}

func matchesRegexList(s string, list []*regexp.Regexp) bool {
	for _, r := range list {
		if r.MatchString(s) {
			return true
		}
	}
	return false
}
