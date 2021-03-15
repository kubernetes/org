package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
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

func removeMembers(memberList []string, configPath string) error {

	var orgs, teams []string

	for _, member := range memberList {
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

				if removed {
					count++
					if info.Name() == "org.yaml" {
						orgs = append(orgs, path)
					}
					if info.Name() == "teams.yaml" {
						teams = append(teams, path)
					}
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		fmt.Print("\n", count)
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

		if dryrun == true {
			return true, nil
		}

		updatedContent := re.ReplaceAll(content, []byte(""))
		err = ioutil.WriteFile(path, updatedContent, 0666)

		if err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil

}
