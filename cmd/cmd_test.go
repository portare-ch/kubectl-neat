/*
Copyright © 2019 Itay Shakury @itaysk

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
package cmd

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func assertErrorNil(err error) bool {
	return err == nil
}
func TestRootCmd(t *testing.T) {
	resourceDataJSONPath := "../test/fixtures/service1-raw.json"
	resourceDataJSONBytes, err := ioutil.ReadFile(resourceDataJSONPath)
	resourceDataJSON := string(resourceDataJSONBytes)
	if err != nil {
		t.Errorf("error readin test data file %s: %v", resourceDataJSONPath, err)
	}
	resourceDataYAMLPath := "../test/fixtures/service1-raw.yaml"
	resourceDataYAMLBytes, err := ioutil.ReadFile(resourceDataYAMLPath)
	resourceDataYAML := string(resourceDataYAMLBytes)
	if err != nil {
		t.Errorf("error readin test data file %s: %v", resourceDataYAMLPath, err)
	}

	testcases := []struct {
		args        []string
		stdin       string
		assertError func(err error) bool
		expOut      string
	}{
		{
			args:        []string{},
			stdin:       "",
			assertError: assertErrorNil,
			expOut:      "",
		},
		{
			args:        []string{},
			stdin:       resourceDataJSON,
			assertError: assertErrorNil,
			expOut:      "apiVersion",
		},
		{
			args:        []string{},
			stdin:       resourceDataYAML,
			assertError: assertErrorNil,
			expOut:      "apiVersion",
		},
		{
			args:        []string{"-f", "-"},
			stdin:       resourceDataJSON,
			assertError: assertErrorNil,
			expOut:      "apiVersion",
		},
		{
			args:  []string{"-f", "/nogood"},
			stdin: "",
			assertError: func(err error) bool {
				_, ok := err.(*os.PathError)
				return ok
			},
			expOut: "",
		},
		{
			args:        []string{"-f", resourceDataJSONPath},
			stdin:       "",
			assertError: assertErrorNil,
			expOut:      "apiVersion",
		},
		{
			args:        []string{"-f", resourceDataYAMLPath},
			stdin:       "",
			assertError: assertErrorNil,
			expOut:      "apiVersion",
		},
	}

	for _, tc := range testcases {
		rootCmd.SetArgs(tc.args)
		if tc.stdin != "" {
			rootCmd.SetIn(bytes.NewReader([]byte(tc.stdin)))
		}
		cmdout := new(bytes.Buffer)
		cmderr := new(bytes.Buffer)
		rootCmd.SetOut(cmdout)
		rootCmd.SetErr(cmderr)
		rootCmd.ParseFlags(tc.args)
		resErr := rootCmd.RunE(rootCmd, tc.args)
		resStdout, err := ioutil.ReadAll(cmdout)
		if err != nil {
			t.Errorf("error reading command output: %v", err)
		}
		resStderr, err := ioutil.ReadAll(cmderr)
		if err != nil {
			t.Errorf("error reading command error: %v\ntest case: %v", err, tc)
		}
		if tc.assertError != nil && !tc.assertError(resErr) {
			t.Errorf("error assertion: have: %#v\ntest case: %v", resErr, tc)
		}
		if !strings.Contains(string(resStdout), tc.expOut) {
			t.Errorf("stdout assertion: have: %s\nwant: %s\ntest case: %v", string(resStdout), tc.expOut, tc)
		}
		if len(resStderr) > 0 {
			t.Errorf("stderr not empty: %s\ntest case: %v", string(resStderr), tc)
		}
	}
}

func TestGetCmd(t *testing.T) {
	kubectl = "../test/kubectl-stub"
	testcases := []struct {
		args        []string
		assertError func(err error) bool
		expOut      string
		expErr      string
	}{
		{
			args: []string{""},
			assertError: func(err error) bool {
				return err != nil && strings.HasPrefix(err.Error(), "Error invoking kubectl")
			},
			expOut: "",
			// kubectl's own error message reaches the user
			expErr: "invalid args",
		},
		{
			args:        []string{"pods"},
			assertError: assertErrorNil,
			expOut:      "apiVersion: v1\nitems:",
			expErr:      "",
		},
		{
			args:        []string{"pods", "mypod"},
			assertError: assertErrorNil,
			expOut:      "apiVersion: v1\nkind: Pod",
			expErr:      "",
		},
		{
			args:        []string{"pods", "mypod", "-o", "yaml"},
			assertError: assertErrorNil,
			expOut:      "apiVersion: v1\nkind: Pod",
			expErr:      "",
		},
		{
			args:        []string{"pods", "mypod", "-o", "json"},
			assertError: assertErrorNil,
			expOut:      `"apiVersion": "v1"`,
			expErr:      "",
		},
		{
			args:        []string{"-ojson", "pods", "mypod"},
			assertError: assertErrorNil,
			expOut:      `"apiVersion": "v1"`,
			expErr:      "",
		},
		{
			args:        []string{"--", "pods", "mypod", "--output=yaml"},
			assertError: assertErrorNil,
			expOut:      "apiVersion: v1\nkind: Pod",
			expErr:      "",
		},
		{
			args: []string{"pods", "mypod", "-o", "wide"},
			assertError: func(err error) bool {
				return err != nil && strings.HasPrefix(err.Error(), "unsupported output format")
			},
			expOut: "",
			expErr: "",
		},
	}

	for _, tc := range testcases {
		rootCmd.SetArgs(tc.args)
		cmdout := new(bytes.Buffer)
		cmderr := new(bytes.Buffer)
		rootCmd.SetOut(cmdout)
		rootCmd.SetErr(cmderr)
		rootCmd.SetIn(bytes.NewReader(nil))
		resErr := getCmd.RunE(getCmd, tc.args)
		resStdout, err := ioutil.ReadAll(cmdout)
		if err != nil {
			t.Errorf("error reading command output: %v", err)
		}
		resStderr, err := ioutil.ReadAll(cmderr)
		if err != nil {
			t.Errorf("error reading command error: %v\ntest case: %v", err, tc)
		}
		if tc.assertError != nil && !tc.assertError(resErr) {
			t.Errorf("error assertion: have: %#v\ntest case: %v", resErr, tc)
		}
		if !strings.Contains(string(resStdout), tc.expOut) {
			t.Errorf("stdout assertion: have: %s\nwant: %s\ntest case: %v", string(resStdout), tc.expOut, tc)
		}
		if tc.expErr == "" && len(resStderr) > 0 {
			t.Errorf("stderr not empty: %s\ntest case: %v", string(resStderr), tc)
		}
		if !strings.Contains(string(resStderr), tc.expErr) {
			t.Errorf("stderr assertion: have: %s\nwant: %s\ntest case: %v", string(resStderr), tc.expErr, tc)
		}
	}
}

func TestParseGetArgs(t *testing.T) {
	testcases := []struct {
		args            []string
		expKubectlArgs  []string
		expOutputFormat string
		expHelp         bool
		expErr          string
	}{
		{args: []string{"secret", "mysecret"}, expKubectlArgs: []string{"secret", "mysecret"}, expOutputFormat: "yaml"},
		{args: []string{"secret", "mysecret", "-n", "other"}, expKubectlArgs: []string{"secret", "mysecret", "-n", "other"}, expOutputFormat: "yaml"},
		{args: []string{"pod", "mypod", "-o", "json"}, expKubectlArgs: []string{"pod", "mypod"}, expOutputFormat: "json"},
		{args: []string{"pod", "mypod", "--output", "json"}, expKubectlArgs: []string{"pod", "mypod"}, expOutputFormat: "json"},
		{args: []string{"pod", "mypod", "--output=json"}, expKubectlArgs: []string{"pod", "mypod"}, expOutputFormat: "json"},
		{args: []string{"pod", "mypod", "-ojson"}, expKubectlArgs: []string{"pod", "mypod"}, expOutputFormat: "json"},
		{args: []string{"pod", "mypod", "-o=json"}, expKubectlArgs: []string{"pod", "mypod"}, expOutputFormat: "json"},
		{args: []string{"pod", "mypod", "-o", "yaml"}, expKubectlArgs: []string{"pod", "mypod"}, expOutputFormat: "yaml"},
		{args: []string{"-o", "json", "--", "svc", "kubernetes", "-n", "default"}, expKubectlArgs: []string{"svc", "kubernetes", "-n", "default"}, expOutputFormat: "json"},
		{args: []string{"--", "pod", "mypod", "-oyaml"}, expKubectlArgs: []string{"pod", "mypod"}, expOutputFormat: "yaml"},
		{args: []string{"deploy", "-l", "app=web", "-A"}, expKubectlArgs: []string{"deploy", "-l", "app=web", "-A"}, expOutputFormat: "yaml"},
		{args: []string{"--help"}, expKubectlArgs: nil, expOutputFormat: "yaml", expHelp: true},
		{args: []string{"pod", "-o", "wide"}, expErr: `unsupported output format "wide"`},
		{args: []string{"pod", "-o", "jsonpath={.metadata.name}"}, expErr: "unsupported output format"},
		{args: []string{"pod", "-o"}, expErr: "flag needs an argument: -o"},
	}
	for _, tc := range testcases {
		kubectlArgs, outFormat, help, err := parseGetArgs(tc.args)
		if tc.expErr != "" {
			if err == nil || !strings.Contains(err.Error(), tc.expErr) {
				t.Errorf("args %q: want error containing %q, have %v", tc.args, tc.expErr, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("args %q: unexpected error: %v", tc.args, err)
			continue
		}
		if strings.Join(kubectlArgs, " ") != strings.Join(tc.expKubectlArgs, " ") {
			t.Errorf("args %q: kubectl args: want %q, have %q", tc.args, tc.expKubectlArgs, kubectlArgs)
		}
		if outFormat != tc.expOutputFormat {
			t.Errorf("args %q: output format: want %q, have %q", tc.args, tc.expOutputFormat, outFormat)
		}
		if help != tc.expHelp {
			t.Errorf("args %q: help: want %v, have %v", tc.args, tc.expHelp, help)
		}
	}
}
