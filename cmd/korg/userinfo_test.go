/*
Copyright 2026 The Kubernetes Authors.

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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-github/v88/github"
	"sigs.k8s.io/prow/pkg/config/org"
)

func TestFindOrgMembership(t *testing.T) {
	configs := map[string]*org.Config{
		"kubernetes": {
			Members: []string{"alice", "BOB"},
			Admins:  []string{"carol"},
		},
		"kubernetes-sigs": {
			Members: []string{"bob"},
			Admins:  []string{"alice"},
		},
		"kubernetes-csi": {},
	}

	cases := map[string][]OrgMembership{
		"alice":  {{Org: "kubernetes", Role: "member"}, {Org: "kubernetes-sigs", Role: "admin"}},
		"bob":    {{Org: "kubernetes", Role: "member"}, {Org: "kubernetes-sigs", Role: "member"}},
		"BoB":    {{Org: "kubernetes", Role: "member"}, {Org: "kubernetes-sigs", Role: "member"}},
		"carol":  {{Org: "kubernetes", Role: "admin"}},
		"nobody": {},
	}

	for user, want := range cases {
		got := findOrgMembership(configs, user)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("user %q: got %+v, want %+v", user, got, want)
		}
	}
}

func TestLoadOrgConfigs(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"kubernetes", "kubernetes-sigs"} {
		dir := filepath.Join(root, "config", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		yaml := fmt.Sprintf("members:\n- user-%s\nadmins:\n- admin-%s\n", name, name)
		if err := os.WriteFile(filepath.Join(dir, "org.yaml"), []byte(yaml), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	configs, err := loadOrgConfigs(root, []string{"kubernetes", "kubernetes-sigs"})
	if err != nil {
		t.Fatalf("loadOrgConfigs: %v", err)
	}
	if got := configs["kubernetes"].Members[0]; got != "user-kubernetes" {
		t.Errorf("kubernetes member: got %q", got)
	}
	if got := configs["kubernetes-sigs"].Admins[0]; got != "admin-kubernetes-sigs" {
		t.Errorf("kubernetes-sigs admin: got %q", got)
	}

	if _, err := loadOrgConfigs(root, []string{"kubernetes-csi"}); err == nil {
		t.Error("expected error for missing org config")
	}
}

func TestSearchOwnerFiles(t *testing.T) {
	const happyBody = `{
		"Results": {
			"kubernetes/kubernetes": {
				"Matches": [
					{"Filename": "OWNERS", "Matches": []},
					{"Filename": "pkg/foo/OWNERS", "Matches": []}
				]
			}
		}
	}`

	tests := []struct {
		name     string
		status   int
		body     string
		wantErr  bool
		wantRepo string
		wantLen  int
	}{
		{name: "happy", status: 200, body: happyBody, wantRepo: "kubernetes/kubernetes", wantLen: 2},
		{name: "500", status: 500, body: "boom", wantErr: true},
		{name: "bad json", status: 200, body: "not json", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var seenURL string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seenURL = r.URL.String()
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			origURL := houndSearchURLOverride
			houndSearchURLOverride = srv.URL
			defer func() { houndSearchURLOverride = origURL }()

			out, err := searchOwnerFiles(context.Background(), srv.Client(), "alice")
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(seenURL, "q=alice") {
				t.Errorf("query missing username: %s", seenURL)
			}
			if got := len(out); got != tc.wantLen {
				t.Errorf("got %d entries, want %d", got, tc.wantLen)
			}
			for _, of := range out {
				if of.Repo != tc.wantRepo {
					t.Errorf("repo: got %q want %q", of.Repo, tc.wantRepo)
				}
				wantURL := "https://github.com/" + of.Repo + "/blob/HEAD/" + of.Path
				if of.URL != wantURL {
					t.Errorf("url: got %q want %q", of.URL, wantURL)
				}
			}
		})
	}
}

func TestFindUserDetails(t *testing.T) {
	ghHits := 0
	gh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ghHits++
		switch {
		case strings.HasSuffix(r.URL.Path, "/users/alice"):
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"login":"alice","company":"Acme Inc"}`)
		case strings.HasSuffix(r.URL.Path, "/users/ghost"):
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message":"Not Found"}`)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer gh.Close()

	hound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Results":{"k/k":{"Matches":[{"Filename":"OWNERS","Matches":[]}]}}}`)
	}))
	defer hound.Close()

	origURL := houndSearchURLOverride
	houndSearchURLOverride = hound.URL
	defer func() { houndSearchURLOverride = origURL }()

	client, err := github.NewClient(github.WithEnterpriseURLs(gh.URL+"/", gh.URL+"/"))
	if err != nil {
		t.Fatalf("github client: %v", err)
	}

	configs := map[string]*org.Config{
		"kubernetes": {Members: []string{"alice"}},
	}

	t.Run("happy", func(t *testing.T) {
		info, err := findUserDetails(context.Background(), client, hound.Client(), configs, "alice")
		if err != nil {
			t.Fatalf("findUserDetails: %v", err)
		}
		if info.Company != "Acme Inc" {
			t.Errorf("company: got %q", info.Company)
		}
		if len(info.Orgs) != 1 || info.Orgs[0].Org != "kubernetes" {
			t.Errorf("orgs: %+v", info.Orgs)
		}
		if len(info.OwnerFiles) != 1 || info.OwnerFiles[0].Repo != "k/k" || info.OwnerFiles[0].Path != "OWNERS" {
			t.Errorf("owner files: %+v", info.OwnerFiles)
		}
		if info.OwnerFiles[0].URL != "https://github.com/k/k/blob/HEAD/OWNERS" {
			t.Errorf("url: %q", info.OwnerFiles[0].URL)
		}
	})

	t.Run("404 user is fatal per-user", func(t *testing.T) {
		_, err := findUserDetails(context.Background(), client, hound.Client(), configs, "ghost")
		if err == nil {
			t.Fatal("expected error for 404 user")
		}
	})

	t.Run("hound failure is non-fatal", func(t *testing.T) {
		badHound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer badHound.Close()
		houndSearchURLOverride = badHound.URL

		info, err := findUserDetails(context.Background(), client, badHound.Client(), configs, "alice")
		if err != nil {
			t.Fatalf("hound failure should not fail user lookup: %v", err)
		}
		if len(info.Warnings) == 0 {
			t.Error("expected warning about hound failure")
		}
		if info.Company != "Acme Inc" {
			t.Errorf("company still expected: %q", info.Company)
		}
	})
}

func TestRunUserinfoJSON(t *testing.T) {
	gh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/users/alice") {
			fmt.Fprint(w, `{"login":"alice","company":"Acme"}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"message":"Not Found"}`)
	}))
	defer gh.Close()

	hound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Results":{}}`)
	}))
	defer hound.Close()

	houndSearchURLOverride = hound.URL
	ghBaseURLOverride = gh.URL + "/"
	defer func() { houndSearchURLOverride = ""; ghBaseURLOverride = "" }()

	root := t.TempDir()
	for _, name := range validOrgs {
		dir := filepath.Join(root, "config", name)
		_ = os.MkdirAll(dir, 0o755)
		_ = os.WriteFile(filepath.Join(dir, "org.yaml"), []byte("members:\n- alice\n"), 0o644)
	}

	var buf bytes.Buffer
	err := runUserinfo(context.Background(), root, []string{"alice"}, true, &buf)
	if err != nil {
		t.Fatalf("runUserinfo: %v", err)
	}

	var got []*UserDetails
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\nraw: %s", err, buf.String())
	}
	if len(got) != 1 || got[0].Username != "alice" || got[0].Company != "Acme" {
		t.Errorf("unexpected: %+v", got)
	}
	if len(got[0].Orgs) != len(validOrgs) {
		t.Errorf("expected membership in %d orgs, got %d", len(validOrgs), len(got[0].Orgs))
	}
}
