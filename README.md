# function-claude-status-transformer
[![CI](https://github.com/upbound/unction-claude-status-transformer/actions/workflows/ci.yml/badge.svg)](https://github.com/upbound/unction-claude-status-transformer/actions/workflows/ci.yml)

Function-claude-status-transformer is a Crossplane Intelligent Function,
specifically designed to help with identifying issues with your Composed
Resources.

Use this function in any Crossplane Composition function pipline where you
would like to have information communicated to end users of your API about
issues with the Compositions.

## Model Support:
|Provider|Models|Notes|
|---|---|---|
|[Anthropic]|[claude-sonnet-4-20250514]|This will be configurable in the future.|

## Using this function
1. Within your Upbound project, run
```
up dep add xpkg.upbound.io/upbound/function-claude-status-transformer:v0.0.0-20250703165412-f44b846b3a
```
2. Within your Composition add a pipeline step that includes the function:
```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: xnetworks.example.upbound.io
spec:
  compositeTypeRef:
    apiVersion: example.upbound.io/v1alpha1
    kind: XNetwork
  mode: Pipeline
  pipeline:
  ...
  - functionRef:
      name: upbound-function-claude-status-transformer
    input:
      apiVersion: function-claude-status-transformer.fn.crossplane.io/v1beta1
      kind: StatusTransformation
      additionalContext: ""
    step: upbound-function-claude-status-transformer
    credentials:
    - name: claude
      source: Secret
      secretRef:
        namespace: crossplane-system
        name: api-key-anthropic
  ...
```
3. Make sure to include a secret for accessing the Claude API, e.g.
```bash
kubectl -n crossplane-system create secret generic api-key-anthropic --from-literal=ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY}"
```

## Building locally

This template uses [Go][go], [Docker][docker], and the [Crossplane CLI][cli] to
build functions.

```shell
# Run code generation - see input/generate.go
$ go generate ./...

# Run tests - see fn_test.go
$ go test ./...

# Build the function's runtime image - see Dockerfile
$ docker build . --tag=runtime

# Build a function package - see package/crossplane.yaml
$ crossplane xpkg build -f package --embed-runtime-image=runtime
```

[functions]: https://docs.crossplane.io/latest/concepts/composition-functions
[go]: https://go.dev
[function guide]: https://docs.crossplane.io/knowledge-base/guides/write-a-composition-function-in-go
[package docs]: https://pkg.go.dev/github.com/crossplane/function-sdk-go
[docker]: https://www.docker.com
[cli]: https://docs.crossplane.io/latest/cli
[Anthropic]: https://docs.anthropic.com/en/docs/about-claude/models/overview
[claude-sonnet-4-20250514]: https://docs.anthropic.com/en/docs/about-claude/models/overview#model-comparison-table
