# kubectl-neat

Remove clutter from Kubernetes manifests to make them more readable.

## Demo

Here is a result of a `kubectl get pod -o yaml` for a simple Pod. The lines marked in red are considered redundant and will be removed from the output by kubectl-neat.

![demo](./demo.png)

## Why

When you create a Kubernetes resource, let's say a Pod, Kubernetes adds a whole bunch of internal system information to the yaml or json that you originally authored. This includes:

- Metadata such as creation timestamp, or some internal IDs
- Fill in for missing attributes with default values
- Additional system attributes created by admission controllers, such as service account token
- Status information

If you try to `kubectl get` resources you have created, they will no longer look like what you originally authored, and will be unreadably verbose.   
`kubectl-neat` cleans up that redundant information for you.

## Installation

### Homebrew (macOS)

This repository is its own Homebrew tap. The formula builds kubectl-neat from source, and Homebrew installs Go as a build dependency if needed.

```bash
brew tap portare-ch/kubectl-neat https://github.com/portare-ch/kubectl-neat
brew trust --formula portare-ch/kubectl-neat/kubectl-neat
brew install kubectl-neat
```

Homebrew only loads formulae from taps outside its own once you trust them; the second line trusts this one formula.

To build the latest commit on `master` instead of the latest release, use `brew install --HEAD kubectl-neat`.

### From source

With Go installed:

```bash
make install                       # installs into /usr/local/bin
make install PREFIX="$HOME/.local" # or into any other directory on your PATH
```

### krew

`kubectl krew install neat` installs the upstream release, which doesn't include the changes made in this fork.

Both Homebrew and `make install` install the `kubectl-neat` executable and a shorter `kneat` alias for it. kubectl also finds `kubectl-neat` as a plugin, so `kubectl neat`, `kubectl-neat` and `kneat` all run the same thing.

## Usage

There are two modes of operation that specify where to get the input document from: a local file or from  Kubernetes.

### Local - file or Stdin

This is the default mode if you run just `kubectl neat`. This command accepts an optional flag `-f/--file` which specifies the file to neat. It can be a path to a local file, or `-` to read the file from stdin. If omitted, it will default to `-`. The file must be a yaml or json file and a valid Kubernetes resource.

There's another optional optional flag, `-o/--output` which specifies the format for the output. If omitted it will default to the same format of the input (auto-detected).

Examples:
```bash
kubectl get pod mypod -o yaml | kubectl neat

kubectl get pod mypod -oyaml | kubectl neat -o json

kubectl neat -f - <./my-pod.json

kubectl neat -f ./my-pod.json

kubectl neat -f ./my-pod.json --output yaml
```

### Kubernetes - kubectl get wrapper

This mode is invoked by calling the `get` subcommand, i.e `kneat get ...`. It is a convenience to run `kubectl get` and then neat the output in a single command. It accepts any argument that `kubectl get` accepts and passes those arguments as is to `kubectl get`, so it uses the same context and namespace kubectl would, including a namespace selected with `kubens`. The output is YAML unless `-o json` is given. Other output formats such as `wide` or `jsonpath` are not supported. Since it executes `kubectl`, it needs to be able to find it in the path.

Examples:
```bash
kneat get secret mysecret
kneat get pod mypod -n kube-system
kneat get svc myservice -o json
kneat get deploy -l app=web
kubectl neat get configmap myconfig
```

The older form with `--`, such as `kubectl neat get -- pod mypod -oyaml`, still works.

# How it works

Besides general tidying for status, metadata, and empty fields, kubectl-neat primarily looks for two types of things: default values inserted by Kubernetes' object model, and common mutating controllers.

## Kubernetes object model defaults

For de-defaulting Kubernetes' object model, we invoke the same code that Kubernetes would have, and see what default values were assigned. If these observed values look like the ones we have in the incoming spec, we conclude they are default. If they weren't, and the user manually set a field to it's default value, it's not a bad thing to remove it anyway.

## Common mutating controllers

Here are the [recommended](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#what-does-each-admission-controller-do) admission controllers, and their relation to kubectl-neat:

controller | description | neat
---|---|---
NamespaceLifecycle | rejects operations on resources in namespaces being deleted | ignore
LimitRanger | set default values for resource requests and limits | ignore
ServiceAccount | set default service account and assign token | Remove `default-token-*` volumes. Remove deprecated `spec.serviceAccount`
TaintNodesByCondition | automatically taint a node based on node conditions | TODO
Priority | validate priority class and add it's value | ignore
DefaultTolerationSeconds | configure pods to temporarily tolarate notready and unreachable taints | TODO
DefaultStorageClass | validate and set default storage class for new pvc | ignore
StorageObjectInUseProtection | prevent deletion of pvc/pv in use by adding a finalizer | ignore
PersistentVolumeClaimResize | enforce pvc resizing only for enabled storage classes | ignore
MutatingAdmissionWebhook | implement the mutating webhook feature | ignore
ValidatingAdmissionWebhook | implement the validating webhook feature | ignore
RuntimeClass | add pod overhead according to runtime class | TODO
ResourceQuota | implement the resource qouta feature | ignore
Kubernetes Scheduler | assign pods to nodes | Remove `spec.nodeName`
