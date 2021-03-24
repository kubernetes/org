package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	flag "github.com/spf13/pflag"
	"k8s.io/test-infra/prow/config/org"
	"sigs.k8s.io/yaml"
)

type options struct {
	confirm    bool
	configPath string
}

func parseOptions() options {
	var o options
	flag.StringVar(&o.configPath, "path", ".", "Path to config directory/subdirectory")
	flag.BoolVar(&o.confirm, "confirm", false, "Modify the actual files or just simulate changes")

	flag.Parse()

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: remove-members [--confirm] [--path] member-file (file-containing-members-list)\n")
		flag.PrintDefaults()
	}

	if len(flag.Args()) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	return o
}

func main() {

	o := parseOptions()

	memberList, err := readMemberList(flag.Args()[0])
	if err != nil {
		log.Fatal(err)
	}

	if err = removeMembers(memberList, o); err != nil {
		log.Fatal(err)
	}

}

//readMemberList reads the list of members to be removed from the given filepath
func readMemberList(path string) ([]string, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var members = strings.Split(string(file), "\n")
	return members, nil
}

//removeMembers walks through the config directory and removes the occurences of the given member name
func removeMembers(memberList []string, o options) error {
	for _, member := range memberList {
		var orgs, teams []string
		fmt.Print(member)

		if err := filepath.Walk(o.configPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			if filepath.Ext(path) != ".yaml" {
				return nil
			}
			removed, removals, err := removeMemberFromFile(member, path, info, o.confirm)
			if err != nil {
				return err
			}

			//Record the org/team name when a member is removed from it
			if removed {
				orgs = append(orgs, removals["orgs"]...)
				teams = append(teams, removals["teams"]...)
			}

			return nil
		}); err != nil {
			return err
		}

		if len(orgs) > 0 {
			fmt.Printf("\n Found %s in %s org(s)", member, strings.Join(orgs, ", "))
		} else {
			fmt.Printf("\n Found %s in no org", member)
		}

		if len(teams) > 0 {
			fmt.Printf("\n Found %s in %s team(s)", member, strings.Join(teams, ", "))
		} else {
			fmt.Printf("\n Found %s in no team", member)
		}

		fmt.Printf("\n Total number of occurences: %d\n", len(orgs)+len(teams))

		//Proceed to committing changes if member is actually removed from somewhere
		if len(orgs)+len(teams) > 0 {
			commitRemovedMembers(member, orgs, teams, o.confirm)
		}
	}

	return nil
}

func removeMemberFromFile(member string, path string, info os.FileInfo, confirm bool) (bool, map[string][]string, error) {

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return false, nil, err
	}

	var cfg org.Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return false, nil, err
	}

	removals := map[string][]string{
		"orgs":  fetchOrgRemovals(cfg, []string{}, member, path),
		"teams": fetchTeamRemovals(cfg.Teams, []string{}, member),
	}

	if len(removals["orgs"])+len(removals["teams"]) > 0 {
		re := regexp.MustCompile(`(\s+)?- ` + member + `(.*)?`)

		matches := re.FindAllIndex(content, -1)

		//Making Sure count from parsed config and regex matches are same
		if len(matches) != len(removals["orgs"])+len(removals["teams"]) {
			log.Printf("\n\n Mismatch in regex count and removal count at %s\n", path)
		}

		if confirm {
			updatedContent := re.ReplaceAll(content, []byte(""))
			if err = ioutil.WriteFile(path, updatedContent, info.Mode()); err != nil {
				return false, removals, err
			}

			cmd := exec.Command("git", "add", path)
			if err := cmd.Run(); err != nil {
				log.Fatal(err)
			}
		}

		return true, removals, nil
	}

	return false, removals, nil

}

func commitRemovedMembers(member string, orgs []string, teams []string, confirm bool) {
	cmd := []string{"echo", "git", "commit"}
	if confirm {
		cmd = cmd[1:]
	}

	orgCommitMsg := "Remove " + member + " from the "
	if len(orgs) == 1 {
		orgCommitMsg += orgs[0] + " org"
		cmd = append(cmd, "-m", orgCommitMsg)
	} else if len(orgs) >= 1 {
		orgCommitMsg += strings.Join(orgs, ", ") + " orgs"
		cmd = append(cmd, "-m", orgCommitMsg)
	}

	teamCommitMsg := "Remove " + member + " from "
	if len(teams) == 1 {
		teamCommitMsg += teams[0] + " team"
		cmd = append(cmd, "-m", teamCommitMsg)
	} else if len(teams) >= 1 {
		teamCommitMsg += strings.Join(teams, ", ") + " teams"
		cmd = append(cmd, "-m", teamCommitMsg)
	}

	e := exec.Command(cmd[0], cmd[1:]...)
	if err := e.Run(); err != nil {
		log.Fatal(err)
	}

}

func fetchOrgRemovals(cfg org.Config, removals []string, member string, path string) []string {
	if cfg.Name != nil {
		removals = fetchRemovals(cfg.Members, removals, member, *cfg.Name)
		removals = fetchRemovals(cfg.Admins, removals, member, *cfg.Name)
	}
	return removals
}

func fetchTeamRemovals(teams map[string]org.Team, removals []string, member string) []string {
	for teamName, v := range teams {
		removals = fetchRemovals(v.Members, removals, member, teamName)
		removals = fetchRemovals(v.Maintainers, removals, member, teamName)
		if len(v.Children) > 0 {
			removals = fetchTeamRemovals(v.Children, removals, member)
		}
	}
	return removals
}

func fetchRemovals(list []string, removals []string, member string, name string) []string {
	for _, i := range list {
		if i == member {
			removals = append(removals, name)
		}
	}
	return removals
}
