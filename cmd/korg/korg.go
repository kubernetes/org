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
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	validOrgs = []string{
		"etcd-io",
		"kubernetes",
		"kubernetes-client",
		"kubernetes-csi",
		"kubernetes-sigs",
	}

	orgConfigPathFormat = "config/%s/org.yaml"

	addHelpText = `
Adds users to GitHub orgs and/or teams

Add user to specified orgs:

	korg add <github username> --org kubernetes --org kubernetes-sigs
	korg add <github username> --org kubernetes,kubernetes-sigs

Note: Adding to teams is currently unsupported.
	`

	removeHelpText = `
Remove users from GitHub orgs

Remove user to specified orgs:

	korg remove <github username> --org kubernetes --org kubernetes-sigs

Note: Removing from teams is currently unsupported.
	`

	auditHelpText = "Audit GitHub org members"

	userinfoHelpText = `
Gets k8s org membership, GitHub profile, and OWNERS file references for user(s).

Honors GITHUB_TOKEN / GH_TOKEN for authenticated GitHub API access.

A failed per-user lookup does not omit that user from the output: in text mode
it renders an ERROR block, and in JSON mode it appears with only "username"
and "error" populated. Check the exit code to detect any failures.

By default, each OWNERS/OWNERS_ALIASES hit is confirmed against the file's
actual content (via the gh CLI) to exclude emeritus-only entries; this costs
one "gh api" call per hit and requires the gh CLI to be installed and
authenticated. Pass --verify-owners=false to skip this and use hound's raw
search hits as-is.

	korg userinfo <github username>
	korg userinfo <github username1> <github username2> <github username3> ...
	korg userinfo --output json <github username>
	korg userinfo --verify-owners=false <github username>
	`
)

type Options struct {
	// global options
	Confirm  bool
	RepoRoot string
	Orgs     []string
	Teams    []string

	// audit options
	AuditOptions
}

func AddMemberToOrgs(username string, options Options) error {
	if invalidOrgs := findInvalidOrgs(options.Orgs); len(invalidOrgs) > 0 {
		return fmt.Errorf("specified invalid orgs: %s", strings.Join(invalidOrgs, ", "))
	}

	if !options.Confirm {
		fmt.Println("!!! running in dry-run mode. pass --confirm to persist changes.")
	}

	configsModified := []string{}
	for _, org := range options.Orgs {
		fmt.Printf("adding %s to %s org\n", username, org)

		relativeConfigPath := fmt.Sprintf(orgConfigPathFormat, org)
		configPath := filepath.Join(options.RepoRoot, relativeConfigPath)

		config, err := readConfig(configPath)
		if err != nil {
			return fmt.Errorf("reading config: %s", err)
		}

		if stringInSliceCaseAgnostic(config.Members, username) || stringInSliceCaseAgnostic(config.Admins, username) {
			return fmt.Errorf("user %s already exists in org %s", username, org)
		}

		newMembers := append(config.Members, username)
		config.Members = newMembers
		caseAgnosticSort(config.Members)

		if options.Confirm {
			fmt.Printf("saving config for %s org\n", org)
			if err := saveConfig(configPath, config); err != nil {
				return fmt.Errorf("saving config: %s", err)
			}
		}

		configsModified = append(configsModified, relativeConfigPath)
	}

	if options.Confirm {
		fmt.Println("committing changes")

		message := fmt.Sprintf("add %s to %s", username, strings.Join(options.Orgs, ", "))
		if err := commitChanges(options.RepoRoot, configsModified, message); err != nil {
			return fmt.Errorf("committing changes: %s", err)
		}
	}
	return nil
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "korg",
		Short: "Manage Kubernetes community owned GitHub organizations",
	}

	o := Options{}
	rootCmd.PersistentFlags().BoolVar(&o.Confirm, "confirm", false, "confirm the changes")
	rootCmd.PersistentFlags().StringVar(&o.RepoRoot, "root", ".", "root of the k/org repo")
	rootCmd.PersistentFlags().StringSliceVar(&o.Orgs, "org", []string{}, "orgs to add the user to")

	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add members to org and/or teams",
		Long:  addHelpText,
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return fmt.Errorf("add only adds one user at a time. specified %d", len(args))
			}

			if len(o.Orgs) == 0 {
				return fmt.Errorf("please specify atleast one org to add the user to")
			}

			if invalidOrgs := findInvalidOrgs(o.Orgs); len(invalidOrgs) > 0 {
				return fmt.Errorf("specified invalid orgs: %s", strings.Join(invalidOrgs, ", "))
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			user := args[0]
			if len(o.Orgs) > 0 {
				return AddMemberToOrgs(user, o)
			}

			return nil
		},
	}

	removeCmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove members from org",
		Long:  removeHelpText,
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if invalidOrgs := findInvalidOrgs(o.Orgs); len(invalidOrgs) > 0 {
				return fmt.Errorf("specified invalid orgs: %s", strings.Join(invalidOrgs, ", "))
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			user := args[0]
			if len(o.Orgs) > 0 {
				return RemoveMemberFromOrgs(o, user)
			}

			return nil
		},
	}

	// korg remove flags
	removeCmd.Flags().StringSliceVar(&o.Orgs, "org", []string{}, "orgs to remove the user from")

	auditCmd := &cobra.Command{
		Use:   "audit",
		Short: "Audit GitHub org members",
		Long:  auditHelpText,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if invalidOrgs := findInvalidOrgs(o.Orgs); len(invalidOrgs) > 0 {
				return fmt.Errorf("specified invalid orgs: %s", strings.Join(invalidOrgs, ", "))
			}

			if o.ActivityThreshold < 0 {
				return fmt.Errorf("activity threshold cannot be negative")
			}

			// TODO: Check if exceptions file is of the right format, if defined

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return OrgAudit(o)
		},
	}

	// korg audit flags
	auditCmd.Flags().IntVar(&o.ActivityThreshold, "activity-threshold", 0, "minimum activity to be considered active. default: 0")
	auditCmd.Flags().StringVar(&o.Period, "period", "y", "period to look back for activity. possible values are defined in https://github.com/cncf/devstats/blob/master/docs/periods.md. default: y (Year)")
	auditCmd.Flags().StringVar(&o.OutputFile, "output-file", "", "parse owners files. default: none")
	auditCmd.Flags().StringVar(&o.ExceptionsFile, "exceptions-file", "", "exceptions for removal. default: none")
	auditCmd.Flags().BoolVar(&o.CheckOwners, "check-owners", false, "parse owners files. default: false")
	auditCmd.Flags().BoolVar(&o.CheckTeams, "check-teams", false, "check which teams the user belongs to. default: false")

	// commands
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(auditCmd)

	var (
		outputFormat string
		verifyOwners bool
	)
	userInfoCmd := &cobra.Command{
		Use:   "userinfo",
		Short: "Get information about user(s)",
		Long:  userinfoHelpText,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch outputFormat {
			case "text", "json":
			default:
				return fmt.Errorf("invalid --output %q (want: text, json)", outputFormat)
			}
			return runUserinfo(cmd.Context(), o.RepoRoot, args, outputFormat == "json", verifyOwners, cmd.OutOrStdout())
		},
	}
	userInfoCmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "output format: text|json")
	userInfoCmd.Flags().BoolVar(&verifyOwners, "verify-owners", true, "confirm each OWNERS/OWNERS_ALIASES hit via the gh CLI and exclude emeritus-only entries; requires gh to be installed and authenticated")
	rootCmd.AddCommand(userInfoCmd)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
