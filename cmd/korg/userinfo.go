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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/google/go-github/v88/github"
	houndclient "github.com/hound-search/hound/client"
	houndindex "github.com/hound-search/hound/index"
	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/prow/pkg/config/org"
	"sigs.k8s.io/yaml"
)

const (
	houndSearchURL      = "https://cs.k8s.io/api/v1/search"
	defaultHTTPTimeout  = 30 * time.Second
	userinfoConcurrency = 4
)

// Test-only overrides. Empty means use defaults / api.github.com.
var (
	houndSearchURLOverride string
	ghBaseURLOverride      string
)

func houndURL() string {
	if houndSearchURLOverride != "" {
		return houndSearchURLOverride
	}
	return houndSearchURL
}

type OrgMembership struct {
	Org  string `json:"org"`
	Role string `json:"role"` // "member" or "admin"
}

type OwnerFile struct {
	Repo string `json:"repo"`
	Path string `json:"path"`
	URL  string `json:"url"`
}

type UserDetails struct {
	Username   string          `json:"username"`
	Company    string          `json:"company,omitempty"`
	Orgs       []OrgMembership `json:"orgs"`
	OwnerFiles []OwnerFile     `json:"owner_files,omitempty"`
	Warnings   []string        `json:"warnings,omitempty"`
	Error      string          `json:"error,omitempty"`
}

// findUserDetails gathers GitHub profile, k8s org membership, and OWNERS file references for a username.
// GitHub or k/org config failures are fatal (returned as error). Hound failures are non-fatal warnings.
// When verifyOwners is true, each OWNERS hit is additionally confirmed against the file's actual
// content (via the gh CLI) to exclude emeritus-only entries; this costs one gh api call per hit.
func findUserDetails(ctx context.Context, gh *github.Client, hc *http.Client, configs map[string]*org.Config, username string, verifyOwners bool) (*UserDetails, error) {
	info := &UserDetails{Username: username}

	u, _, err := gh.Users.Get(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("github user %q: %w", username, err)
	}
	if u.Company != nil {
		info.Company = *u.Company
	}

	info.Orgs = findOrgMembership(configs, username)

	hits, err := searchOwnerFiles(ctx, hc, username)
	switch {
	case err != nil:
		info.Warnings = append(info.Warnings, fmt.Sprintf("OWNERS lookup failed: %v", err))
	case !verifyOwners:
		info.OwnerFiles = hits
	default:
		info.OwnerFiles = make([]OwnerFile, 0, len(hits))
		for _, hit := range hits {
			content, err := fetchOwnerFileContent(ctx, hit.Repo, hit.Path)
			if err != nil {
				// Fail open: we can't confirm the hit, but hound already found it, so keep it.
				info.Warnings = append(info.Warnings, fmt.Sprintf("could not verify %s/%s: %v", hit.Repo, hit.Path, err))
				info.OwnerFiles = append(info.OwnerFiles, hit)
				continue
			}
			if isActiveOwner(content, hit.Path, username) {
				info.OwnerFiles = append(info.OwnerFiles, hit)
			}
		}
	}

	return info, nil
}

func findOrgMembership(configs map[string]*org.Config, username string) []OrgMembership {
	out := []OrgMembership{}
	orgNames := make([]string, 0, len(configs))
	for name := range configs {
		orgNames = append(orgNames, name)
	}
	sort.Strings(orgNames)

	for _, name := range orgNames {
		cfg := configs[name]
		switch {
		case stringInSliceCaseAgnostic(cfg.Admins, username):
			out = append(out, OrgMembership{Org: name, Role: "admin"})
		case stringInSliceCaseAgnostic(cfg.Members, username):
			out = append(out, OrgMembership{Org: name, Role: "member"})
		}
	}
	return out
}

func loadOrgConfigs(repoRoot string, orgs []string) (map[string]*org.Config, error) {
	out := make(map[string]*org.Config, len(orgs))
	for _, name := range orgs {
		path := filepath.Join(repoRoot, fmt.Sprintf(orgConfigPathFormat, name))
		cfg, err := readConfig(path)
		if err != nil {
			return nil, fmt.Errorf("loading org config %s: %w", name, err)
		}
		out[name] = cfg
	}
	return out, nil
}

func ownerFileURL(repo, path string) string {
	return fmt.Sprintf("https://github.com/%s/blob/HEAD/%s", repo, path)
}

func searchOwnerFiles(ctx context.Context, hc *http.Client, username string) ([]OwnerFile, error) {
	q := url.Values{}
	// OWNERS/OWNERS_ALIASES entries are always YAML list items, so anchor the
	// search on "- <username>" to skip files that merely mention the name in
	// prose or as a substring of another username.
	q.Set("q", "- "+username)
	q.Set("literal", "true")
	q.Set("repos", "*")
	q.Set("rng", ":20")
	q.Set("files", "OWNERS(_ALIASES)?")
	q.Set("excludeFiles", "vendor/")
	q.Set("i", "true")
	q.Set("stats", "true")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, houndURL()+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return nil, fmt.Errorf("hound %s: %s", resp.Status, bytes.TrimSpace(snippet))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var r houndclient.Response
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("decoding hound response: %w", err)
	}

	out := []OwnerFile{}
	for repo, matches := range r.Results {
		if matches == nil {
			continue
		}
		for _, fm := range matches.Matches {
			if !hasRealMatch(fm.Matches, username) {
				continue
			}
			out = append(out, OwnerFile{
				Repo: repo,
				Path: fm.Filename,
				URL:  ownerFileURL(repo, fm.Filename),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Repo != out[j].Repo {
			return out[i].Repo < out[j].Repo
		}
		return out[i].Path < out[j].Path
	})
	return out, nil
}

// hasRealMatch reports whether any hit line is an uncommented YAML list item whose
// value is exactly username, rather than a comment or a substring match like
// "- alice-bot" or "- alicexyz" (hound's search is substring-based, so those also
// match a query for "- alice").
func hasRealMatch(matches []*houndindex.Match, username string) bool {
	for _, m := range matches {
		line := strings.TrimSpace(m.Line)
		rest, ok := strings.CutPrefix(line, "-")
		if !ok {
			continue
		}
		rest = strings.Trim(strings.TrimSpace(rest), `"'`)
		if strings.EqualFold(rest, username) {
			return true
		}
	}
	return false
}

// ownersFileContent models the fields of an OWNERS file relevant to deciding
// whether a username is a current (non-emeritus) approver or reviewer.
type ownersFileContent struct {
	Approvers         []string                    `json:"approvers,omitempty"`
	Reviewers         []string                    `json:"reviewers,omitempty"`
	EmeritusApprovers []string                    `json:"emeritus_approvers,omitempty"`
	EmeritusReviewers []string                    `json:"emeritus_reviewers,omitempty"`
	Filters           map[string]ownersFilterRule `json:"filters,omitempty"`
}

type ownersFilterRule struct {
	Approvers []string `json:"approvers,omitempty"`
	Reviewers []string `json:"reviewers,omitempty"`
}

// ownersAliasesContent models an OWNERS_ALIASES file.
type ownersAliasesContent struct {
	Aliases map[string][]string `json:"aliases,omitempty"`
}

// runGHCLI invokes the gh CLI and returns its stdout. Overridden in tests.
var runGHCLI = func(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

// fetchOwnerFileContent fetches the raw contents of an OWNERS or OWNERS_ALIASES file at repo HEAD
// via the gh CLI, so it reuses the caller's existing `gh auth login` session.
func fetchOwnerFileContent(ctx context.Context, repo, path string) ([]byte, error) {
	endpoint := fmt.Sprintf("repos/%s/contents/%s", repo, escapeGitHubPath(path))
	return runGHCLI(ctx, "api", endpoint, "-H", "Accept: application/vnd.github.raw")
}

// escapeGitHubPath percent-encodes each path segment while preserving "/" separators.
func escapeGitHubPath(path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}

// isActiveOwner reports whether username is a current (non-emeritus) approver,
// reviewer, or alias member per the given OWNERS/OWNERS_ALIASES file content.
// It fails open (returns true) if content can't be parsed, since hound already
// found the hit and we'd rather over- than under-report.
func isActiveOwner(content []byte, path, username string) bool {
	if filepath.Base(path) == "OWNERS_ALIASES" {
		var a ownersAliasesContent
		if err := yaml.Unmarshal(content, &a); err != nil {
			return true
		}
		for _, members := range a.Aliases {
			if stringInSliceCaseAgnostic(members, username) {
				return true
			}
		}
		return false
	}

	var o ownersFileContent
	if err := yaml.Unmarshal(content, &o); err != nil {
		return true
	}
	if stringInSliceCaseAgnostic(o.Approvers, username) || stringInSliceCaseAgnostic(o.Reviewers, username) {
		return true
	}
	for _, f := range o.Filters {
		if stringInSliceCaseAgnostic(f.Approvers, username) || stringInSliceCaseAgnostic(f.Reviewers, username) {
			return true
		}
	}
	return false
}

// renderText writes a human-readable form of UserDetails to w.
func (u *UserDetails) renderText(w io.Writer) {
	fmt.Fprintf(w, "\n=== %s\n", u.Username)
	if u.Company != "" {
		fmt.Fprintln(w, "Company:", u.Company)
	} else {
		fmt.Fprintln(w, "Company: **Not Found**")
	}

	fmt.Fprintln(w, "Orgs:")
	if len(u.Orgs) == 0 {
		fmt.Fprintln(w, "  (none)")
	}
	for _, m := range u.Orgs {
		fmt.Fprintf(w, "  %s (%s)\n", m.Org, m.Role)
	}

	if len(u.OwnerFiles) > 0 {
		fmt.Fprintln(w, "Owner Files:")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  REPO\tPATH\tURL")
		for _, of := range u.OwnerFiles {
			fmt.Fprintf(tw, "  %s\t%s\t%s\n", of.Repo, of.Path, of.URL)
		}
		tw.Flush()
	}

	for _, warn := range u.Warnings {
		fmt.Fprintln(w, "Warning:", warn)
	}
}

// newGitHubClient returns a GitHub client authenticated via GITHUB_TOKEN or GH_TOKEN if set.
func newGitHubClient(httpClient *http.Client) (*github.Client, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	opts := []github.ClientOptionsFunc{}
	if httpClient != nil {
		opts = append(opts, github.WithHTTPClient(httpClient))
	}
	if token != "" {
		opts = append(opts, github.WithAuthToken(token))
	}
	if ghBaseURLOverride != "" {
		opts = append(opts, github.WithEnterpriseURLs(ghBaseURLOverride, ghBaseURLOverride))
	}
	return github.NewClient(opts...)
}

// runUserinfo fetches info for every username concurrently and writes ordered output to w.
// Returns a joined error of all per-user failures; users without errors still render.
func runUserinfo(ctx context.Context, repoRoot string, usernames []string, outputJSON, verifyOwners bool, w io.Writer) error {
	configs, err := loadOrgConfigs(repoRoot, validOrgs)
	if err != nil {
		return err
	}

	hc := &http.Client{Timeout: defaultHTTPTimeout}
	gh, err := newGitHubClient(hc)
	if err != nil {
		return fmt.Errorf("building github client: %w", err)
	}

	results := make([]*UserDetails, len(usernames))
	errs := make([]error, len(usernames))

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(userinfoConcurrency)
	for i, name := range usernames {
		g.Go(func() error {
			info, err := findUserDetails(gctx, gh, hc, configs, name, verifyOwners)
			if err != nil {
				errs[i] = err
				return nil
			}
			results[i] = info
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("unexpected errgroup error: %w", err)
	}

	if outputJSON {
		out := make([]*UserDetails, len(results))
		for i, r := range results {
			switch {
			case r != nil:
				out[i] = r
			case errs[i] != nil:
				out[i] = &UserDetails{Username: usernames[i], Error: errs[i].Error()}
			default:
				out[i] = &UserDetails{Username: usernames[i], Error: "unknown error"}
			}
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(out); err != nil {
			return err
		}
	} else {
		for i, r := range results {
			if r != nil {
				r.renderText(w)
			}
			if errs[i] != nil {
				fmt.Fprintf(w, "\n=== %s\nERROR: %v\n", usernames[i], errs[i])
			}
		}
	}

	return errors.Join(errs...)
}
