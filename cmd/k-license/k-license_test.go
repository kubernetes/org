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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var optTest = &Options{}

// Running E2E Test
func TestAddLicense(t *testing.T) {
	cases := []struct {
		confirm bool
		desc    string
	}{
		{
			confirm: true,
			desc:    "confirm=true, files should actually change",
		},
		{
			confirm: false,
			desc:    "confirm=false, files should not actually change",
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			optTest.templatesDir = "../../hack/boilerplate"
			optTest.excludeDirs = excludeDirsLocations
			optTest.confirm = c.confirm
			fmt.Printf("Running test case with %s", c.desc)
			tmpTestDir := createTmpDir()
			optTest.path = tmpTestDir
			if err := optTest.Run(); err != nil {
				t.Errorf("error running test case %s: %v", c.desc, err)
			}
			defer os.RemoveAll(tmpTestDir)
		})
	}
}

func createTmpDir() string {
	tmpDir, err := os.MkdirTemp("", "testdata")
	if err != nil {
		panic(err)
	}

	err = filepath.Walk("testdata", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			destPath := filepath.Join(tmpDir, strings.TrimPrefix(path, "testdata"+string(filepath.Separator)))

			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}

			srcFile, err := os.Open(path)
			if err != nil {
				return err
			}
			defer srcFile.Close()

			destFile, err := os.Create(destPath)
			if err != nil {
				return err
			}
			defer destFile.Close()

			if _, err := io.Copy(destFile, srcFile); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		panic(err)
	}

	return tmpDir
}
