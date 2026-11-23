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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"unicode"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
)

var outputFormat *string
var inputFile *string

func init() {
	outputFormat = rootCmd.PersistentFlags().StringP("output", "o", "yaml", "output format: yaml or json")
	inputFile = rootCmd.Flags().StringP("file", "f", "-", "file path to neat, or - to read from stdin")
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	rootCmd.MarkFlagFilename("file")
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(versionCmd)
}

// Execute is the entry point for the command package
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use: "kubectl-neat",
	Example: `kubectl get pod mypod -o yaml | kubectl neat
kubectl get pod mypod -oyaml | kubectl neat -o json
kubectl neat -f - <./my-pod.json
kubectl neat -f ./my-pod.json
kubectl neat -f ./my-pod.json --output yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var in, out []byte
		var err error
		if *inputFile == "-" {
			stdin := cmd.InOrStdin()
			in, err = ioutil.ReadAll(stdin)
		} else {
			in, err = ioutil.ReadFile(*inputFile)
			if err != nil {
				return err
			}
		}
		outFormat := *outputFormat
		if !cmd.Flag("output").Changed {
			outFormat = "same"
		}
		out, err = NeatYAMLOrJSON(in, outFormat)
		if err != nil {
			return err
		}
		cmd.Print(string(out))
		return nil
	},
}

var kubectl string = "kubectl"

var getCmd = &cobra.Command{
	Use:   "get [kubectl get arguments]",
	Short: "Run kubectl get and print the neated result",
	Long: `Run "kubectl get" with the given arguments and print the neated result.

All arguments are handed to kubectl get unchanged, so the namespace, context and
kubeconfig that kubectl would use apply here too, including a namespace selected
with kubens. The output is YAML unless -o json is given.`,
	Example: `kubectl neat get secret mysecret
kubectl neat get pod mypod -n kube-system
kubectl neat get svc myservice -o json
kubectl neat get deploy -l app=web`,
	// kubectl get's flags are not known here, so all arguments are taken as they
	// are and only -o/--output and -h/--help are picked out by parseGetArgs.
	DisableFlagParsing: true,
	// A failing kubectl get prints its own error, the usage text adds nothing.
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		kubectlArgs, outFormat, help, err := parseGetArgs(args)
		if err != nil {
			return err
		}
		if help {
			return cmd.Help()
		}
		// kubectl-neat works on json, so that is what kubectl is always asked for.
		kubectlCmd := exec.Command(kubectl, append([]string{"get", "-o", "json"}, kubectlArgs...)...)
		kubectlCmd.Stdin = cmd.InOrStdin()
		// kubectl's errors and warnings go to the user, and stay out of the json.
		kubectlCmd.Stderr = cmd.ErrOrStderr()
		kres, err := kubectlCmd.Output()
		if err != nil {
			return fmt.Errorf("Error invoking kubectl as %v: %v", kubectlCmd.Args, err)
		}
		out, err := NeatYAMLOrJSON(kres, outFormat)
		if err != nil {
			return err
		}
		if !bytes.HasSuffix(out, []byte("\n")) {
			out = append(out, '\n')
		}
		cmd.Print(string(out))
		return nil
	},
}

// parseGetArgs splits the arguments of the get subcommand into the arguments for
// kubectl get and the output format of kubectl-neat, which is yaml unless
// -o/--output says otherwise. The output flag is taken out of the kubectl
// arguments, since kubectl itself must always print json. A "--" is accepted
// and dropped, so the older form "kubectl neat get -- pod mypod" still works.
func parseGetArgs(args []string) (kubectlArgs []string, outFormat string, help bool, err error) {
	outFormat = "yaml"
	seenSeparator := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--" && !seenSeparator:
			seenSeparator = true
		case arg == "-h" || arg == "--help":
			help = true
		case arg == "-o" || arg == "--output":
			if i+1 == len(args) {
				return nil, "", false, fmt.Errorf("flag needs an argument: %s", arg)
			}
			i++
			outFormat = args[i]
		case strings.HasPrefix(arg, "--output="):
			outFormat = strings.TrimPrefix(arg, "--output=")
		case strings.HasPrefix(arg, "-o"):
			outFormat = strings.TrimPrefix(strings.TrimPrefix(arg, "-o"), "=")
		default:
			kubectlArgs = append(kubectlArgs, arg)
		}
	}
	if outFormat != "yaml" && outFormat != "json" {
		return nil, "", false, fmt.Errorf("unsupported output format %q: kubectl-neat prints yaml or json", outFormat)
	}
	return kubectlArgs, outFormat, help, nil
}

// populated by goreleaser
var (
	Version = "v0.0.0+unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print kubectl-neat version",
	Long:  "Print the version of kubectl-neat",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("kubectl-neat version: %s\n", Version)
	},
}

func isJSON(s []byte) bool {
	return bytes.HasPrefix(bytes.TrimLeftFunc(s, unicode.IsSpace), []byte{'{'})
}

// NeatYAMLOrJSON converts 'in' to json if needed, invokes neat, and converts back if needed according the the outputFormat argument: yaml/json/same
func NeatYAMLOrJSON(in []byte, outputFormat string) (out []byte, err error) {
	var injson, outjson string
	itsYaml := !isJSON(in)
	if itsYaml {
		injsonbytes, err := yaml.YAMLToJSON(in)
		if err != nil {
			return nil, fmt.Errorf("error converting from yaml to json : %v", err)
		}
		injson = string(injsonbytes)
	} else {
		injson = string(in)
	}

	outjson, err = Neat(injson)
	if err != nil {
		return nil, fmt.Errorf("error neating : %v", err)
	}

	if outputFormat == "yaml" || (outputFormat == "same" && itsYaml) {
		out, err = yaml.JSONToYAML([]byte(outjson))
		if err != nil {
			return nil, fmt.Errorf("error converting from json to yaml : %v", err)
		}
	} else {
		out = []byte(outjson)
	}
	return
}
