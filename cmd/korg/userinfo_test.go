/*
Copyright The Kubernetes Authors.

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
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-github/v88/github"
	"sigs.k8s.io/prow/pkg/config/org"
)

// stubGHCLI overrides runGHCLI for the duration of t, restoring it on cleanup.
func stubGHCLI(t *testing.T, fn func(ctx context.Context, args ...string) ([]byte, error)) {
	t.Helper()
	orig := runGHCLI
	runGHCLI = fn
	t.Cleanup(func() { runGHCLI = orig })
}

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
					{"Filename": "OWNERS", "Matches": [{"Line": "  - alice", "LineNumber": 3}]},
					{"Filename": "pkg/foo/OWNERS", "Matches": [{"Line": "  - alice", "LineNumber": 5}]},
					{"Filename": "commented/OWNERS", "Matches": [{"Line": "  # - alice", "LineNumber": 2}]},
					{"Filename": "substring/OWNERS", "Matches": [{"Line": "  - alice-bot", "LineNumber": 7}]}
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
			parsed, perr := url.Parse(seenURL)
			if perr != nil {
				t.Fatalf("parsing seen URL: %v", perr)
			}
			if got := parsed.Query().Get("q"); got != "- alice" {
				t.Errorf("query: got %q, want %q", got, "- alice")
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
	stubGHCLI(t, func(ctx context.Context, args ...string) ([]byte, error) {
		if len(args) == 4 && args[0] == "api" && args[1] == "repos/k/k/contents/OWNERS" {
			return []byte("approvers:\n- alice\n"), nil
		}
		return nil, fmt.Errorf("unexpected gh invocation: %v", args)
	})

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
		fmt.Fprint(w, `{"Results":{"k/k":{"Matches":[{"Filename":"OWNERS","Matches":[{"Line":"- alice","LineNumber":2}]}]}}}`)
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
		info, err := findUserDetails(context.Background(), client, hound.Client(), configs, "alice", true)
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
		_, err := findUserDetails(context.Background(), client, hound.Client(), configs, "ghost", true)
		if err == nil {
			t.Fatal("expected error for 404 user")
		}
	})

	t.Run("hound failure is non-fatal", func(t *testing.T) {
		badHound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer badHound.Close()
		origHoundURL := houndSearchURLOverride
		houndSearchURLOverride = badHound.URL
		defer func() { houndSearchURLOverride = origHoundURL }()

		info, err := findUserDetails(context.Background(), client, badHound.Client(), configs, "alice", true)
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

	t.Run("verifyOwners false skips gh CLI verification", func(t *testing.T) {
		calls := 0
		stubGHCLI(t, func(ctx context.Context, args ...string) ([]byte, error) {
			calls++
			return nil, fmt.Errorf("gh CLI should not be invoked when verifyOwners is false")
		})

		info, err := findUserDetails(context.Background(), client, hound.Client(), configs, "alice", false)
		if err != nil {
			t.Fatalf("findUserDetails: %v", err)
		}
		if calls != 0 {
			t.Errorf("expected 0 gh CLI invocations, got %d", calls)
		}
		if len(info.OwnerFiles) != 1 || info.OwnerFiles[0].Path != "OWNERS" {
			t.Errorf("expected raw hound hit to pass through unverified, got %+v", info.OwnerFiles)
		}
	})
}

func TestIsActiveOwner(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		content string
		want    bool
	}{
		{
			name:    "current approver",
			path:    "OWNERS",
			content: "approvers:\n- alice\n",
			want:    true,
		},
		{
			name:    "current reviewer",
			path:    "OWNERS",
			content: "reviewers:\n- alice\n",
			want:    true,
		},
		{
			name:    "filter approver",
			path:    "OWNERS",
			content: "filters:\n  \".*\\\\.go$\":\n    approvers:\n    - alice\n",
			want:    true,
		},
		{
			name:    "emeritus approver only",
			path:    "OWNERS",
			content: "emeritus_approvers:\n- alice\n",
			want:    false,
		},
		{
			name:    "emeritus reviewer only",
			path:    "OWNERS",
			content: "emeritus_reviewers:\n- alice\n",
			want:    false,
		},
		{
			name:    "not present at all",
			path:    "OWNERS",
			content: "approvers:\n- bob\n",
			want:    false,
		},
		{
			name:    "alias member",
			path:    "OWNERS_ALIASES",
			content: "aliases:\n  sig-foo-approvers:\n  - alice\n",
			want:    true,
		},
		{
			name:    "not in any alias",
			path:    "OWNERS_ALIASES",
			content: "aliases:\n  sig-foo-approvers:\n  - bob\n",
			want:    false,
		},
		{
			name:    "unparseable content fails open",
			path:    "OWNERS",
			content: "not: [valid",
			want:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isActiveOwner([]byte(tc.content), tc.path, "alice"); got != tc.want {
				t.Errorf("isActiveOwner(%q, %q) = %v, want %v", tc.path, tc.content, got, tc.want)
			}
		})
	}
}

func TestFindUserDetailsExcludesEmeritusOwner(t *testing.T) {
	stubGHCLI(t, func(ctx context.Context, args ...string) ([]byte, error) {
		if len(args) == 4 && args[0] == "api" && args[1] == "repos/k/k/contents/OWNERS" {
			return []byte("emeritus_approvers:\n- alice\n"), nil
		}
		return nil, fmt.Errorf("unexpected gh invocation: %v", args)
	})

	gh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/users/alice"):
			fmt.Fprint(w, `{"login":"alice"}`)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer gh.Close()

	hound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Results":{"k/k":{"Matches":[{"Filename":"OWNERS","Matches":[{"Line":"- alice","LineNumber":2}]}]}}}`)
	}))
	defer hound.Close()

	origURL := houndSearchURLOverride
	houndSearchURLOverride = hound.URL
	defer func() { houndSearchURLOverride = origURL }()

	client, err := github.NewClient(github.WithEnterpriseURLs(gh.URL+"/", gh.URL+"/"))
	if err != nil {
		t.Fatalf("github client: %v", err)
	}

	info, err := findUserDetails(context.Background(), client, hound.Client(), map[string]*org.Config{}, "alice", true)
	if err != nil {
		t.Fatalf("findUserDetails: %v", err)
	}
	if len(info.OwnerFiles) != 0 {
		t.Errorf("expected emeritus-only hit to be excluded, got %+v", info.OwnerFiles)
	}
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
	err := runUserinfo(context.Background(), root, []string{"alice"}, true, true, &buf)
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

func TestRunUserinfoJSONIncludesFailedUsers(t *testing.T) {
	gh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		_ = os.WriteFile(filepath.Join(dir, "org.yaml"), []byte("members: []\n"), 0o644)
	}

	var buf bytes.Buffer
	err := runUserinfo(context.Background(), root, []string{"ghost"}, true, true, &buf)
	if err == nil {
		t.Fatal("expected error for missing user")
	}

	var got []*UserDetails
	if unmarshalErr := json.Unmarshal(buf.Bytes(), &got); unmarshalErr != nil {
		t.Fatalf("decode: %v\nraw: %s", unmarshalErr, buf.String())
	}
	if len(got) != 1 || got[0].Username != "ghost" {
		t.Fatalf("unexpected: %+v", got)
	}
	if got[0].Error == "" {
		t.Error("expected non-empty error for failed lookup")
	}
}

func TestRunUserinfoText(t *testing.T) {
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
		fmt.Fprint(w, `{"Results":{"k/k":{"Matches":[{"Filename":"OWNERS","Matches":[{"Line":"- alice","LineNumber":2}]}]}}}`)
	}))
	defer hound.Close()

	houndSearchURLOverride = hound.URL
	ghBaseURLOverride = gh.URL + "/"
	defer func() { houndSearchURLOverride = ""; ghBaseURLOverride = "" }()

	stubGHCLI(t, func(ctx context.Context, args ...string) ([]byte, error) {
		return []byte("approvers:\n- alice\n"), nil
	})

	root := t.TempDir()
	for _, name := range validOrgs {
		dir := filepath.Join(root, "config", name)
		_ = os.MkdirAll(dir, 0o755)
		body := "members: []\n"
		if name == "kubernetes" {
			body = "members:\n- alice\n"
		}
		_ = os.WriteFile(filepath.Join(dir, "org.yaml"), []byte(body), 0o644)
	}

	var buf bytes.Buffer
	if err := runUserinfo(context.Background(), root, []string{"alice"}, false, true, &buf); err != nil {
		t.Fatalf("runUserinfo: %v", err)
	}

	out := buf.String()
	for _, want := range []string{
		"=== alice",
		"Company: Acme",
		"Orgs:",
		"  kubernetes (member)",
		"Owner Files:",
		"REPO",
		"PATH",
		"URL",
		"k/k",
		"OWNERS",
		"https://github.com/k/k/blob/HEAD/OWNERS",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q, got:\n%s", want, out)
		}
	}
}

func TestRunUserinfoJSONMixedBatch(t *testing.T) {
	gh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/users/alice"):
			fmt.Fprint(w, `{"login":"alice","company":"Acme"}`)
		case strings.HasSuffix(r.URL.Path, "/users/bob"):
			fmt.Fprint(w, `{"login":"bob","company":"Corp"}`)
		case strings.HasSuffix(r.URL.Path, "/users/ghost"):
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message":"Not Found"}`)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
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
		_ = os.WriteFile(filepath.Join(dir, "org.yaml"), []byte("members: []\n"), 0o644)
	}

	usernames := []string{"alice", "ghost", "bob"}
	var buf bytes.Buffer
	err := runUserinfo(context.Background(), root, usernames, true, true, &buf)
	if err == nil {
		t.Fatal("expected a joined error for the failed user")
	}

	var got []*UserDetails
	if unmarshalErr := json.Unmarshal(buf.Bytes(), &got); unmarshalErr != nil {
		t.Fatalf("decode: %v\nraw: %s", unmarshalErr, buf.String())
	}
	if len(got) != len(usernames) {
		t.Fatalf("got %d entries, want %d", len(got), len(usernames))
	}

	for i, name := range usernames {
		if got[i].Username != name {
			t.Errorf("entry %d: got username %q, want %q (order not preserved)", i, got[i].Username, name)
		}
	}
	if got[0].Company != "Acme" || got[0].Error != "" {
		t.Errorf("alice: got %+v", got[0])
	}
	if got[1].Error == "" || got[1].Company != "" {
		t.Errorf("ghost: expected error-only entry, got %+v", got[1])
	}
	if got[2].Company != "Corp" || got[2].Error != "" {
		t.Errorf("bob: got %+v", got[2])
	}
}
