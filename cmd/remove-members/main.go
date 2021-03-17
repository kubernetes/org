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

var dryrun bool = true
var configPath string

func main() {
	flag.StringVar(&configPath, "path", ".", "Path to config directory/subdirectory")
	flag.BoolVar(&dryrun, "dryrun", true, "Enable Dryrun or not. Dryrun simulates changes to be applied and prints removal details")

	flag.Parse()

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: remove-members [--dryrun] [--path] member-file (file-containing-members-list)\n")
		flag.PrintDefaults()
	}

	if len(flag.Args()) != 1 {
		flag.Usage()
		os.Exit(0)
	}

	memberList, err := readMemberList(flag.Args()[0])
	if err != nil {
		log.Fatal(err)
	}

	if err = removeMembers(memberList, configPath); err != nil {
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
func removeMembers(memberList []string, configPath string) error {
	for _, member := range memberList {
		var orgs, teams []string
		count := 0
		fmt.Print(member)

		if err := filepath.Walk(configPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			if matched, err := filepath.Match("*.yaml", filepath.Base(path)); err != nil {
				return err
			} else if matched {
				removed, removeCount, err := removeMemberFromFile(member, path, info)
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
			commitRemovedMembers(member, orgs, teams)
		}
	}

	return nil
}

func removeMemberFromFile(member string, path string, info os.FileInfo) (bool, int, error) {

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return false, 0, err
	}

	re := regexp.MustCompile(`(\s+)?- ` + member + `(.*)?`)

	matches := re.FindAllIndex(content, -1)

	if len(matches) >= 1 {

		//Mofify the file only if it's not a dry run
		if dryrun {
			return true, len(matches), nil
		}

		updatedContent := re.ReplaceAll(content, []byte(""))
		if err = ioutil.WriteFile(path, updatedContent, info.Mode()); err != nil {
			return false, len(matches), err
		}

		cmd := exec.Command("git", "add", path)
		if err := cmd.Run(); err != nil {
			log.Fatal(err)
		}

		return true, len(matches), nil
	}

	return false, len(matches), nil

}

func commitRemovedMembers(member string, orgs []string, teams []string) {
	cmd := []string{"commit"}

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

	fmt.Printf("\nCommit Command: %q\n\n", strings.Join(cmd, " "))

	//Execute the git command only if not a dry run
	if !dryrun {
		cmd := exec.Command("git", cmd...)
		if err := cmd.Run(); err != nil {
			log.Fatal(err)
		}
	}
}
