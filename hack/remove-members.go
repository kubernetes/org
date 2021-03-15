package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var dryrun bool
var repoRoot string

func main() {
	flag.StringVar(&repoRoot, "root", ".", "Root of the repository")
	flag.BoolVar(&dryrun, "dryrun", true, "Enable Dryrun or not")

	flag.Parse()

	memberList, err := readMemberList(flag.Args()[0])
	if err != nil {
		log.Fatal(err)
	}

	configPath := repoRoot + "/config"

	err = removeMembers(memberList, configPath)
	if err != nil {
		log.Fatal(err)
	}

}

//readMemberList reads the list of members to be removed from the given filepath
func readMemberList(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var members []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		members = append(members, scanner.Text())
	}
	return members, scanner.Err()
}

//removeMembers walks through the config directory and removes the occurences of the given member name
func removeMembers(memberList []string, configPath string) error {
	for _, member := range memberList {
		var orgs, teams []string
		count := 0
		fmt.Print(member)

		err := filepath.Walk(configPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			if matched, err := filepath.Match("*.yaml", filepath.Base(path)); err != nil {
				return err
			} else if matched {
				removed, err := removeMemberFromFile(member, path)
				if err != nil {
					return err
				}

				//Record the org/team name when a member is removed from it
				if removed {
					count++
					if info.Name() == "org.yaml" {
						orgs = append(orgs, filepath.Base(filepath.Dir(path)))
					}
					if info.Name() == "teams.yaml" {
						teams = append(teams, filepath.Base(filepath.Dir(path)))
					}
				}
			}
			return nil
		})
		if err != nil {
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

func removeMemberFromFile(member string, path string) (bool, error) {

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return false, err
	}

	re := regexp.MustCompile(`(\s+)?- ` + member + `(.*)?`)

	if re.Match(content) {

		//Mofify the file only if it's not a dry run
		if dryrun == true {
			return true, nil
		}

		updatedContent := re.ReplaceAll(content, []byte(""))
		err = ioutil.WriteFile(path, updatedContent, 0666)

		if err != nil {
			return false, err
		}

		cmd := exec.Command("git", "add", path)
		err := cmd.Run()

		if err != nil {
			log.Fatal(err)
		}

		return true, nil
	}

	return false, nil

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
		err := cmd.Run()

		if err != nil {
			log.Fatal(err)
		}
	}
}
