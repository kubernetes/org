package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	flag "github.com/spf13/pflag"
)

type options struct {
	confirm    bool
	configPath string
}

func parseOptions() options {
	var o options
	flag.StringVar(&o.configPath, "path", ".", "Path to config directory/subdirectory")
	flag.BoolVar(&o.confirm, "confirm", true, "Modify the actual files or just simulate changes")

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
		count := 0
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
			removed, removeCount, err := removeMemberFromFile(member, path, info, o.confirm)
			if err != nil {
				return err
			}

			//Record the org/team name when a member is removed from it
			if removed {
				count += removeCount
				if info.Name() == "org.yaml" {
					orgs = append(orgs, filepath.Base(filepath.Dir(path)))
				}
				if info.Name() == "teams.yaml" {
					teams = append(teams, filepath.Base(filepath.Dir(path)))
				}
			}

			return nil
		}); err != nil {
			return err
		}

		sort.Strings(orgs)
		sort.Strings(teams)

		fmt.Printf("\n Orgs: %v\n Teams: %v\n Number of occurences: %d\n", orgs, teams, count)

		//Proceed to committing changes if member is actually removed from somewhere
		if count > 0 {
			commitRemovedMembers(member, orgs, teams, o.confirm)
		}
	}

	return nil
}

func removeMemberFromFile(member string, path string, info os.FileInfo, confirm bool) (bool, int, error) {

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return false, 0, err
	}

	re := regexp.MustCompile(`(\s+)?- ` + member + `(.*)?`)

	matches := re.FindAllIndex(content, -1)

	if len(matches) >= 1 {

		if confirm {
			updatedContent := re.ReplaceAll(content, []byte(""))
			if err = ioutil.WriteFile(path, updatedContent, info.Mode()); err != nil {
				return false, len(matches), err
			}

			cmd := exec.Command("git", "add", path)
			if err := cmd.Run(); err != nil {
				log.Fatal(err)
			}
		}

		return true, len(matches), nil
	}

	return false, len(matches), nil

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
