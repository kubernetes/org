/*
Copyright 2023 The Kubernetes Authors.

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
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/olekukonko/tablewriter"
)

type Contribution struct {
	Rank         int
	Username     string
	ContribCount int
	Orgs         []string
}

type Values struct {
	Items [][]interface{} `json:"values,omitempty"`
}

type Frames struct {
	Schema map[string]interface{} `json:"schema,omitempty"`
	Data   Values                 `json:"data,omitempty"`
}

type DevStatsRequest struct {
	Queries []Query `json:"queries"`
}

type Query struct {
	RefID        string `json:"refId"`
	DatasourceID int    `json:"datasourceId"`
	RawSQL       string `json:"rawSql"`
	Format       string `json:"format"`
}

type AuditOptions struct {
	Period            string
	ActivityThreshold int
	OutputFile        string
	ExceptionsFile    string
	CheckOwners       bool
	CheckTeams        bool
}

type UserInfo struct {
	Username      string
	Contributions int
	Orgs          []string
	Teams         map[string][]string
	IsOwner       bool
}

type Exception struct {
	Username string
	Reason   string
}

func GetAllUsersInOrgs(o Options, orgs []string) (map[string]UserInfo, error) {
	users := make(map[string]UserInfo)
	config, err := LoadOrgs(o)
	if err != nil {
		return nil, err
	}

	for _, org := range orgs {
		for _, u := range append(config[org].Members, config[org].Admins...) {
			if user, found := users[strings.ToLower(u)]; found {
				if user.Orgs == nil {
					user.Orgs = []string{}
				}
				user.Orgs = append(user.Orgs, org)
				users[strings.ToLower(u)] = user
			} else {
				users[strings.ToLower(u)] = UserInfo{
					Username: u,
					Orgs:     []string{org},
				}
			}
		}

		for teamName, team := range config[org].Teams {
			for _, u := range append(team.Members, team.Maintainers...) {
				if user, found := users[strings.ToLower(u)]; found {
					if user.Teams == nil {
						user.Teams = make(map[string][]string)
					}
					user.Teams[org] = append(user.Teams[org], teamName)
					users[strings.ToLower(u)] = user
				} else {
					users[strings.ToLower(u)] = UserInfo{
						Username: u,
						Teams:    map[string][]string{org: {teamName}},
					}
				}
			}
		}
	}

	return users, nil
}

func GetContributions(period string) (map[string]Contribution, error) {
	postBody := DevStatsRequest{
		Queries: []Query{
			{
				RefID:        "A",
				DatasourceID: 1,
				RawSQL: fmt.Sprintf(`select
  sub."Rank",
  sub.name as name,
  sub.value
from (
  select row_number() over (order by sum(value) desc) as "Rank",
    split_part(name, '$$$', 1) as name,
    sum(value) as value
  from
    shdev
  where
    series = 'hdev_contributionsallall'
    and period = '%s'
  group by
    split_part(name, '$$$', 1)
) sub`, period),
				Format: "table",
			},
		},
	}

	requestBody, err := json.Marshal(postBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post("https://k8s.devstats.cncf.io/api/ds/query", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad error code from devstats: %d: %w", resp.StatusCode, err)
	}

	var parsed map[string]map[string]map[string][]Frames
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &parsed)
	if err != nil {
		return nil, fmt.Errorf("unable to parse json from devstats: %w", err)
	}

	ranks := parsed["results"]["A"]["frames"][0].Data.Items[0]
	usernames := parsed["results"]["A"]["frames"][0].Data.Items[1]
	contribCounts := parsed["results"]["A"]["frames"][0].Data.Items[2]

	contribs := make(map[string]Contribution)
	for i := 0; i < len(ranks); i++ {
		username := usernames[i].(string)
		contribs[username] = Contribution{
			int(ranks[i].(float64)),
			username,
			int(contribCounts[i].(float64)),
			[]string{},
		}
	}
	return contribs, nil
}

func ReadExceptions(filepath string) ([]Exception, error) {
	var exceptions []Exception

	data, err := os.ReadFile(filepath)
	if err != nil {
		return exceptions, err
	}

	r := csv.NewReader(bytes.NewBuffer(data))
	records, err := r.ReadAll()
	if err != nil {
		return exceptions, err
	}

	for _, record := range records[1:] {
		exceptions = append(exceptions, Exception{Username: record[0], Reason: record[1]})
	}

	return exceptions, nil
}

func usernameNotInContributors(contribs map[string]Contribution, username string) bool {
	_, found := contribs[username]

	return !found
}

func usernameBelowActivityThreshold(contribs map[string]Contribution, username string, activityThreshold int) bool {
	if !usernameNotInContributors(contribs, username) {
		return false
	}

	if contribs[strings.ToLower(username)].ContribCount <= activityThreshold {
		return true
	}

	return false
}

func usernameInExceptions(exceptionalUsers []string, username string) bool {
	for _, exceptionalUser := range exceptionalUsers {
		if exceptionalUser == username {
			return true
		}
	}

	return false
}

func OrgAudit(o Options) error {
	fmt.Printf("Running analysis with a lookback period of %s and activity threshold of %d\n", o.Period, o.ActivityThreshold)

	var exceptionalUsers []string
	if o.ExceptionsFile != "" {
		fmt.Printf("reading exceptions from %s\n", o.ExceptionsFile)
		exceptions, err := ReadExceptions(o.ExceptionsFile)
		if err != nil {
			return err
		}

		// build indexable map for exceptions
		for _, exception := range exceptions {
			exceptionalUsers = append(exceptionalUsers, exception.Username)
		}

		// Print exceptions to stdout
		fmt.Println("Total Exceptions:", len(exceptions))
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Username", "Reason"})
		for _, exception := range exceptions {
			table.Append([]string{exception.Username, exception.Reason})
		}
		table.Render()
	}

	fmt.Println("fetching data from devstats")
	contributions, err := GetContributions(o.Period)
	if err != nil {
		return err
	}

	fmt.Println("total contributors:", len(contributions))

	fmt.Println("fetching org members")
	users, err := GetAllUsersInOrgs(o, validOrgs)
	if err != nil {
		return err
	}

	fmt.Println("filtering org members")
	var orgMembersBelowThresholdAfterException []UserInfo
	for _, userInfo := range users {
		if usernameInExceptions(exceptionalUsers, userInfo.Username) {
			fmt.Printf("username %s in exceptions. skipping...\n", userInfo.Username)
			continue
		}
		if usernameNotInContributors(contributions, userInfo.Username) ||
			usernameBelowActivityThreshold(contributions, userInfo.Username, o.ActivityThreshold) {

			userInfo.Contributions = contributions[userInfo.Username].ContribCount
			orgMembersBelowThresholdAfterException = append(orgMembersBelowThresholdAfterException, userInfo)

			fmt.Println("user below threshold or not in devstats:", userInfo.Username, " contributions: ", contributions[userInfo.Username].ContribCount)
		}
	}

	// sort users for readability
	sort.Slice(orgMembersBelowThresholdAfterException, func(i, j int) bool {
		return orgMembersBelowThresholdAfterException[i].Username < orgMembersBelowThresholdAfterException[j].Username
	})

	// populate if member is an owner
	if o.CheckOwners {
		for i, member := range orgMembersBelowThresholdAfterException {
			isOwner, err := IsOwner(member.Username)
			if err != nil {
				return err
			}
			fmt.Printf("checking if user %s is owner: %v\n", member.Username, isOwner)
			member.IsOwner = isOwner
			orgMembersBelowThresholdAfterException[i] = member
		}
	}

	fmt.Println("Total \"Org Members\":", len(users))
	fmt.Println("Total \"Org Members\" below threshold after exceptions:", len(orgMembersBelowThresholdAfterException))

	f, err := os.Create(o.OutputFile)
	if err != nil {
		return err
	}

	w := bufio.NewWriter(f)
	table := tablewriter.NewWriter(w)
	table.SetAutoWrapText(false)

	headers := []string{"Username", "Orgs"}
	if o.CheckTeams {
		headers = append(headers, "Teams")
	}
	if o.CheckOwners {
		headers = append(headers, "Owner", "Owners Link")
	}
	table.SetHeader(headers)

	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")

	for _, v := range orgMembersBelowThresholdAfterException {
		row := []string{
			v.Username,
			strings.Join(v.Orgs, ", "),
		}

		if o.CheckTeams {
			teams := []string{}
			for org, team := range v.Teams {
				teams = append(teams, fmt.Sprintf("%s: %s", org, strings.Join(team, ", ")))
			}
			row = append(row, strings.Join(teams, "; "))
		}

		if o.CheckOwners {
			if v.IsOwner {
				row = append(row, "yes", fmt.Sprintf("https://go.k8s.io/owners/%s", v.Username))
			} else {
				row = append(row, "no", "")
			}
		}

		table.Append(row)
	}
	table.Render()

	w.Flush()

	return nil
}
