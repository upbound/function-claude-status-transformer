/*
Copyright 2025 The Upbound Authors.

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
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/bedrock"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/tidwall/gjson"
	"google.golang.org/protobuf/encoding/protojson"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/response"

	"github.com/upbound/function-claude-status-transformer/input/v1beta1"
	canthropic "github.com/upbound/function-claude-status-transformer/internal/credentials/anthropic"
	caws "github.com/upbound/function-claude-status-transformer/internal/credentials/aws"
)

const system = `
You are a Kubernetes operator trained in identifying and fixing issues with 
Kubernetes resources. You will be given a set of composed Kubernetes resources, 
some of which may be in a bad state. Your job is to identify the issues present 
on each of the resources in the context of the full set of resources and 
communicate these issues to the user.
`

const prompt = `
<instructions>
Please follow these instructions carefully:

1. Analyze the provided set of composed resources searching for any resources
   that are in an unhealthy state. Use the status.conditions field to determine
   the status of the resource.

2. For each resource that is in an unhealthy state, provide a succinct,
   human-readable explanation of the issue. Only include resources that are 
   unhealthy. Use the metadata.name and namespace field to identify the 
   resource.

3. If there are no unhealthy resources, output an empty "resourceStatuses"
   array, an "overallStatus" of "Ready", and a summary of "No unhealthy
   resources found".

4. Along with the set of composed resources, I will also provide you with the
   last status you. If your summary matches the previous summary and/or
   the resource status messages are still accurate within the status reason,
   return the previous status unchanged.

5. For each explanation, provide a JSON object with the structure shown below in
   the <example> tag. Submit the JSON object to the submit_status tool.
</instructions>

<example>
{
	"resourceStatuses": [{
		"name": [resource-name],
		"namespace": [resource-namespace],
		"kind": [resource-kind],
		"apiVersion": [resource-apiVersion],
		"ready": [true|false],
		"message": [human-friendly-explanation-of-problems]
	}],
	"overallStatus": ["Ready"|"NotReady],
	"summary": [summary-of-problems]
}
</example>
`

const vars = `
Here are the newly composed resources:

<composite>
{{ .Composite }}
</composite>

If there are any existing composed resources, they will be provided here:

<composed>
{{ .Composed }}
</composed>

The last status you produced is provided here:
<last-status>
{{ .LastStatus }}
</last-status>

Additional input provided by the Kubernetes operator is provided here:

<input>
{{ .Input }}
</input>
`

const (
	submitStatusToolName             = "submit_status"
	submitStatusToolSchemaProperties = `{"status_json":{"type": "string","description":"The status object, represented in JSON, to submit"}}`
	submitStatusToolDescription      = `
Accepts a JSON object containing a status object. Must be valid JSON in the
shape supplied in the <example> tag.
`
)

const (
	conditionTypeClaudeHealthy xpv1.ConditionType = "HealthyAccordingToClaude"
)

var marshaler = protojson.MarshalOptions{
	UseProtoNames:   true,
	EmitUnpopulated: false,
	UseEnumNumbers:  false,
}

// composedResourceStatus is the status of a composed resource as reported by
// Claude. It contains the name of the resource, whether it's ready (which
// should always be false), and a human-readable explanation of the problems.
type composedResourceStatus struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace,omitempty"`
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	Ready      bool   `json:"ready"`
	Message    string `json:"message"`
}

// CompositionStatus is the status of the composition as reported by Claude. It
// contains the status of each composed resource, the overall status of the
// composition, and a summary of the problems.
type CompositionStatus struct {
	ResourceStatuses []composedResourceStatus `json:"resourceStatuses"`
	OverallStatus    string                   `json:"overallStatus"`
	Summary          string                   `json:"summary"`
}

// Variables used to form the prompt.
type Variables struct {
	// Observed composite resource, as a YAML manifest.
	Composite string

	// Observed composed resources, as a stream of YAML manifests.
	Composed string

	// Last status you produced.
	LastStatus string

	// Input - i.e. user prompt.
	Input string
}

// Function asks Claude to compose resources.
type Function struct {
	fnv1.UnimplementedFunctionRunnerServiceServer

	vars *template.Template
	log  logging.Logger

	c client.Client
}

type Option func(*Function)

func WithClient(c client.Client) Option {
	return func(f *Function) {
		f.c = c
	}
}

// NewFunction creates a new function powered by Claude.
func NewFunction(log logging.Logger, opts ...Option) *Function {
	f := &Function{
		log:  log,
		vars: template.Must(template.New("vars").Parse(vars)),
	}

	for _, o := range opts {
		o(f)
	}
	return f
}

// RunFunction runs the Function.
func (f *Function) RunFunction(ctx context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error) { //nolint:gocyclo // TODO(negz): Factor out the API calling bits.
	log := f.log.WithValues("tag", req.GetMeta().GetTag())
	log.Info("Running function", "tag", req.GetMeta().GetTag())

	rsp := response.To(req, response.DefaultTTL)

	in := &v1beta1.StatusTransformation{}
	if err := request.GetInput(req, in); err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot get Function input from %T", req))
		return rsp, nil
	}

	xr, err := marshaler.Marshal(req.GetObserved().GetComposite())
	if err != nil {
		response.Fatal(rsp, errors.Wrap(err, "cannot convert observed XR to YAML"))
		return rsp, nil
	}

	cds, err := ProtoMapToJSON(req.GetObserved().GetResources())
	if err != nil {
		response.Fatal(rsp, errors.Wrap(err, "cannot convert observed composed resources to YAML"))
		return rsp, nil
	}

	lastStatus, err := lastStatusFromObserved(req)
	if err != nil {
		response.Fatal(rsp, errors.Wrap(err, "cannot get last status from observed"))
		return rsp, nil
	}

	lastStatusJSON, err := json.Marshal(lastStatus)
	if err != nil {
		response.Fatal(rsp, errors.Wrap(err, "cannot marshal last status to JSON"))
		return rsp, nil
	}

	vars := &strings.Builder{}
	if err := f.vars.Execute(vars, &Variables{Composite: string(xr), Composed: cds, Input: in.AdditionalContext, LastStatus: string(lastStatusJSON)}); err != nil {
		response.Fatal(rsp, errors.Wrap(err, "cannot build prompt from template"))
		return rsp, nil
	}

	log.Debug("Using prompt", "prompt", vars.String())

	client, err := f.getClient(ctx, in, req)
	if err != nil {
		response.Fatal(rsp, errors.Wrap(err, "cannot get LLM client"))
	}

	messages := []anthropic.MessageParam{
		{
			Role: anthropic.MessageParamRoleUser,
			Content: []anthropic.ContentBlockParamUnion{
				{
					OfText: &anthropic.TextBlockParam{
						Text:         prompt,
						CacheControl: anthropic.NewCacheControlEphemeralParam(),
					},
				},
			},
		},
		{
			Role: anthropic.MessageParamRoleUser,
			Content: []anthropic.ContentBlockParamUnion{
				{
					OfText: &anthropic.TextBlockParam{
						Text: vars.String(),
					},
				},
			},
		},
	}

	model := getModel(in)
	for {
		message, err := client.Messages.New(ctx, anthropic.MessageNewParams{
			MaxTokens: 1024,
			Model:     model,
			System: []anthropic.TextBlockParam{
				{
					Text:         system,
					CacheControl: anthropic.NewCacheControlEphemeralParam(),
				},
			},
			Temperature: param.Opt[float64]{Value: 0}, // As little randomness as possible.
			Tools: []anthropic.ToolUnionParam{
				{
					OfTool: &anthropic.ToolParam{
						Name:        submitStatusToolName,
						Description: anthropic.String(submitStatusToolDescription),
						InputSchema: anthropic.ToolInputSchemaParam{
							Properties: map[string]any{
								"status_stream": map[string]any{
									"type":        "string",
									"description": "The status stream, represented in JSON, to submit",
								},
							},
						},
					},
				},
			},
			Messages: messages,
		})
		if err != nil {
			response.Fatal(rsp, errors.Wrapf(err, "cannot message Claude"))
			return rsp, nil
		}

		// Save Claude's response, to feed back to it on the next call.
		messages = append(messages, message.ToParam())

		toolResults := []anthropic.ContentBlockParamUnion{}
		for _, block := range message.Content {
			switch block.AsAny().(type) {

			// This could happen several times, as Claude calls the
			// tool to check whether its YAML is valid.
			case anthropic.ToolUseBlock:
				log.Debug("Got tool use block from Claude", "tool_name", block.Name, "tool_input", block.JSON.Input.Raw())

				switch block.Name {
				case submitStatusToolName:
					y := gjson.Get(block.JSON.Input.Raw(), "status_stream").String()
					if y == "" {
						response.Fatal(rsp, errors.Errorf("Claude didn't provide 'status_stream' input property for %q tool", block.Name))
						return rsp, nil
					}

					var status CompositionStatus
					result := ""
					if err := json.Unmarshal([]byte(y), &status); err != nil {
						result = err.Error()
					} else {
						log.Debug("Received composition status from Claude",
							"overallStatus", status.OverallStatus,
							"summary", status.Summary,
							"resourceCount", len(status.ResourceStatuses))

						jsonStatuses, err := json.Marshal(status.ResourceStatuses)
						if err != nil {
							response.Fatal(rsp, errors.Wrap(err, "cannot marshal resource statuses to JSON"))
							return rsp, nil
						}

						cond := &fnv1.Condition{
							Type:    string(conditionTypeClaudeHealthy),
							Message: &status.Summary,
							Reason:  string(jsonStatuses),
						}

						if status.OverallStatus == "Ready" {
							cond.Status = fnv1.Status_STATUS_CONDITION_TRUE
						} else {
							cond.Status = fnv1.Status_STATUS_CONDITION_FALSE
						}

						rsp.Conditions = append(rsp.Conditions, cond)

						rsp.Results = append(rsp.Results, &fnv1.Result{
							Severity: fnv1.Severity_SEVERITY_NORMAL,
							Message:  status.Summary,
							Reason:   ptr.To(string(jsonStatuses)),
						})

						return rsp, nil
					}

					log.Debug("Submitted status stream", "result", result, "isError", result != "")
					toolResults = append(toolResults, anthropic.NewToolResultBlock(block.ID, result, result != ""))

				default:
					response.Fatal(rsp, errors.Errorf("Claude tried to use unknown tool %q", block.Name))
					return rsp, nil
				}

			// Despite the prompt, Claude insists on sending a text
			// message explaining what it's going to do before it
			// calls the tool. So this could be called several
			// times, and only sometimes with YAML.
			case anthropic.TextBlock:
				log.Debug("Received text block from Claude", "text", block.Text)
			}
		}

		// Claude's done using tools.
		if len(toolResults) == 0 {
			break
		}

		// Claude's not done using tools. Send the messages again, this
		// time with the tool results.
		messages = append(messages, anthropic.NewUserMessage(toolResults...))
	}

	// We should never get here.
	response.Fatal(rsp, errors.New("Claude didn't return a YAML stream of composed resource manifests"))
	return rsp, nil
}

// ProtoMapToJSON converts a map of string keys to proto messages into a single JSON string
// suitable for LLM consumption. Returns the JSON-encoded string of the entire map.
func ProtoMapToJSON(protoMap map[string]*fnv1.Resource) (string, error) {
	result := make(map[string]interface{})

	for key, protoMsg := range protoMap {
		if protoMsg == nil {
			result[key] = nil
			continue
		}

		jsonBytes, err := marshaler.Marshal(protoMsg)
		if err != nil {
			return "", fmt.Errorf("failed to marshal proto message for key '%s': %w", key, err)
		}

		// Unmarshal JSON bytes back into interface{} using the keys output by
		// the proto marshaler.
		var jsonObj interface{}
		if err := json.Unmarshal(jsonBytes, &jsonObj); err != nil {
			return "", fmt.Errorf("failed to unmarshal JSON for key '%s': %w", key, err)
		}

		result[key] = jsonObj
	}

	// Marshal the entire map to a single JSON string
	finalJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal final JSON: %w", err)
	}

	return string(finalJSON), nil
}

func lastStatusFromObserved(req *fnv1.RunFunctionRequest) (CompositionStatus, error) {
	oxr, err := request.GetObservedCompositeResource(req)
	if err != nil {
		return CompositionStatus{}, errors.Wrap(err, "cannot get observed composite resource")
	}

	cond := oxr.Resource.GetCondition(conditionTypeClaudeHealthy)

	status := CompositionStatus{
		Summary:          cond.Message,
		ResourceStatuses: []composedResourceStatus{},
	}

	if err := json.Unmarshal([]byte(cond.Reason), &status.ResourceStatuses); err != nil {
		return CompositionStatus{}, errors.Wrap(err, "cannot unmarshal resource statuses from condition reason")
	}

	if cond.Status == corev1.ConditionTrue {
		status.OverallStatus = "Ready"
	} else {
		status.OverallStatus = "NotReady"
	}

	return status, nil
}

const (
	defaultAWSRegion       = "us-east-1"
	defaultAWSBedrockModel = "us.anthropic.claude-sonnet-4-20250514-v1:0"
)

// getClient returns an anthropic.Client configured to either use Anthropic's
// APIs directly, using a standard API key, or AWS Bedrock which uses AWS
// authentication methods (including PRODIC from Upbound).
func (f *Function) getClient(ctx context.Context, in *v1beta1.StatusTransformation, req *fnv1.RunFunctionRequest) (anthropic.Client, error) {
	if in.UseAWS() {
		// Ensure the region is default if it's not provided.
		if len(in.AWS.Region) == 0 {
			in.AWS.Region = defaultAWSRegion
		}

		a := caws.New(f.c, in)
		cfg, err := a.GetConfig(ctx)
		if err != nil {
			return anthropic.Client{}, errors.Wrap(err, "failed to derive AWS Config from the environment")
		}

		return anthropic.NewClient(bedrock.WithConfig(*cfg)), nil
	}

	a := canthropic.New(req)
	key, err := a.GetAPIKey()
	if err != nil {
		return anthropic.Client{}, errors.Wrap(err, "failed to retrieve Anthropic API key")
	}

	return anthropic.NewClient(option.WithAPIKey(key)), nil
}

// getModel returns the anthropic.Model that should be used with the incoming
// request. In the event of using AWS, we ensure the model is defaulted
// correctly.
func getModel(in *v1beta1.StatusTransformation) anthropic.Model {
	if in.UseAWS() {
		if len(in.AWS.Bedrock.ModelID) == 0 {
			in.AWS.Bedrock.ModelID = defaultAWSBedrockModel
		}
		return anthropic.Model(in.AWS.Bedrock.ModelID)
	}
	return anthropic.ModelClaudeSonnet4_0
}
