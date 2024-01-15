package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/mux"
	p "github.com/poom90914/pulumi_golang/intenal/pulumi"
	"github.com/pulumi/pulumi/sdk/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/go/common/workspace"
	"github.com/pulumi/pulumi/sdk/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
)

var project = "pulumi_over_http"

type CreateSiteReq struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

type UpdateSiteReq struct {
	Content string `json:"content"`
}

type SiteResponse struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type ListSitesResponse struct {
	IDs []string `json:"ids"`
}

// creates new sites
func createHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var createReq CreateSiteReq
	err := json.NewDecoder(req.Body).Decode(&createReq)
	if err != nil {
		w.WriteHeader(400)
		fmt.Fprintf(w, "failed to parse create request")
		return
	}

	ctx := context.Background()

	stackName := createReq.ID
	program := p.CreatePulumiProgram(createReq.Content)

	s, err := auto.NewStackInlineSource(ctx, stackName, project, program)
	if err != nil {
		// if stack already exists, 409
		if auto.IsCreateStack409Error(err) {
			w.WriteHeader(409)
			fmt.Fprintf(w, fmt.Sprintf("stack %q already exists", stackName))
			return
		}

		w.WriteHeader(500)
		fmt.Fprintf(w, err.Error())
		return
	}
	s.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: "us-west-2"})

	// deploy the stack
	// we'll write all of the update logs to st	out so we can watch requests get processed
	upRes, err := s.Up(ctx, optup.ProgressStreams(os.Stdout))
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, err.Error())
		return
	}

	response := &SiteResponse{
		ID:  stackName,
		URL: upRes.Outputs["websiteUrl"].Value.(string),
	}
	json.NewEncoder(w).Encode(&response)
}

// lists all sites
func listHandler(c *gin.Context) {
	ctx := context.Background()
	// set up a workspace with only enough information for the list stack operations

	ws, err := auto.NewLocalWorkspace(ctx, auto.Project(workspace.Project{
		Name:    tokens.PackageName(project),
		Runtime: workspace.NewProjectRuntimeInfo("go", nil),
	}))
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}
	stacks, err := ws.ListStacks(ctx)
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}
	var ids []string
	for _, stack := range stacks {
		ids = append(ids, stack.Name)
	}

	response := &ListSitesResponse{
		IDs: ids,
	}
	c.JSON(500, gin.H{
		"message": &response,
	})
}

// gets info about a specific site
func getHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(req)
	stackName := params["id"]
	// we don't need a program since we're just getting stack outputs
	var program pulumi.RunFunc = nil
	ctx := context.Background()
	s, err := auto.SelectStackInlineSource(ctx, stackName, project, program)
	c.JSON(500, gin.H{
		"message": &response,
	})
	if err != nil {
		// if the stack doesn't already exist, 404
		if auto.IsSelectStack404Error(err) {
			w.WriteHeader(404)
			fmt.Fprintf(w, fmt.Sprintf("stack %q not found", stackName))
			return
		}

		w.WriteHeader(500)
		fmt.Fprintf(w, err.Error())
		return
	}

	// fetch the outputs from the stack
	outs, err := s.Outputs(ctx)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, err.Error())
		return
	}

	response := &SiteResponse{
		ID:  stackName,
		URL: outs["websiteUrl"].Value.(string),
	}
	json.NewEncoder(w).Encode(&response)
}

// updates the content for an existing site
func updateHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var updateReq UpdateSiteReq
	err := json.NewDecoder(req.Body).Decode(&updateReq)
	if err != nil {
		w.WriteHeader(400)
		fmt.Fprintf(w, "failed to parse create request")
		return
	}

	ctx := context.Background()
	params := mux.Vars(req)
	stackName := params["id"]
	program := p.CreatePulumiProgram(updateReq.Content)

	s, err := auto.SelectStackInlineSource(ctx, stackName, project, program)
	if err != nil {
		if auto.IsSelectStack404Error(err) {
			w.WriteHeader(404)
			fmt.Fprintf(w, fmt.Sprintf("stack %q not found", stackName))
			return
		}

		w.WriteHeader(500)
		fmt.Fprintf(w, err.Error())
		return
	}
	s.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: "us-west-2"})

	// deploy the stack
	// we'll write all of the update logs to st	out so we can watch requests get processed
	upRes, err := s.Up(ctx, optup.ProgressStreams(os.Stdout))
	if err != nil {
		// if we already have another update in progress, return a 409
		if auto.IsConcurrentUpdateError(err) {
			w.WriteHeader(409)
			fmt.Fprintf(w, "stack %q already has update in progress", stackName)
			return
		}

		w.WriteHeader(500)
		fmt.Fprintf(w, err.Error())
		return
	}

	response := &SiteResponse{
		ID:  stackName,
		URL: upRes.Outputs["websiteUrl"].Value.(string),
	}
	json.NewEncoder(w).Encode(&response)
}

// deletes a site
func deleteHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx := context.Background()
	params := mux.Vars(req)
	stackName := params["id"]
	// program doesn't matter for destroying a stack
	program := p.CreatePulumiProgram("")

	s, err := auto.SelectStackInlineSource(ctx, stackName, project, program)
	if err != nil {
		// if stack doesn't already exist, 404
		if auto.IsSelectStack404Error(err) {
			w.WriteHeader(404)
			fmt.Fprintf(w, fmt.Sprintf("stack %q not found", stackName))
			return
		}

		w.WriteHeader(500)
		fmt.Fprintf(w, err.Error())
		return
	}
	s.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: "us-west-2"})

	// destroy the stack
	// we'll write all of the logs to stdout so we can watch requests get processed
	_, err = s.Destroy(ctx, optdestroy.ProgressStreams(os.Stdout))
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, err.Error())
		return
	}

	// delete the stack and all associated history and config
	err = s.Workspace().RemoveStack(ctx, stackName)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, err.Error())
		return
	}

	w.WriteHeader(200)
}
