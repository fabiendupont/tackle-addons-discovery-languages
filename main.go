package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	hub "github.com/konveyor/tackle-hub/addon"
	"github.com/konveyor/tackle-hub/model"
	"os"
	"os/exec"
	"strconv"
)

const (
	Kind = "Tag"
)

var (
	// addon adapter.
	addon = hub.Addon
)

//
// Data Addon data passed in the secret.
// TODO: Replace Git* fields with fetching the info from the hub
type Data struct {
	Application uint   `json:"application"`
	GitURL      string `json:"git_url"`
	GitBranch   string `json:"git_branch"`
	GitPath     string `json:"git_path"`
}

//
// main
func main() {
	//
	// Task update: This addon has started.
	_ = addon.Started()

	//
	// Get the addon data associated with the task.
	d := &Data{}
	_ = addon.DataWith(d)

	//
	// Validate the addon data and enforce defaults
	if d.Application == 0 {
		_ = addon.Failed("field 'application' is missing in addon data")
		os.Exit(1)
	}
	if d.GitURL == "" {
		_ = addon.Failed("field 'git_url' is missing in addon data")
		os.Exit(1)
	}

	//
	// Clone Git repository
	cloneGitRepository(d)

	//
	// Get the main language
	language := getLanguage(d)

	// Set language tag for the application
	tag(d, language)

	// Task update: The addon has succeeded
	_ = addon.Succeeded()
}

//
// Clone Git repository
// TODO: Add support for non anonymous Git operations
// TODO: Add support for fetching credentials from Hub
func cloneGitRepository(d *Data) {
	_ = addon.Activity("cloning Git repository")

	gitCloneOptions := &git.CloneOptions{
		URL:               d.GitURL,
		ReferenceName:     plumbing.ReferenceName(d.GitBranch),
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	}

	_, err := git.PlainClone("/tmp/app", false, gitCloneOptions)
	if err != nil {
		_ = addon.Failed(fmt.Sprintf("failed to clone the Git repository: %v", err))
		os.Exit(1)
	}

	return
}

//
// Returns the most represented language in the repository
func getLanguage(d *Data) (language string) {
	_ = addon.Activity("identifying languages in the repository")

	cmd := exec.Command(
		"/usr/local/bin/github-linguist",
		"/tmp/app/"+d.GitPath,
		"--json",
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		_ = addon.Failed(fmt.Sprintf("failed to identify languages in the repository: %s", stderr.String()))
		os.Exit(1)
	}

	// Read the JSON output into a map
	type statistic struct {
		Size       string `json:"size"`
		Percentage string `json:"percentage"`
	}
	report := map[string]statistic{}
	_ = json.Unmarshal([]byte(stdout.String()), &report)
	lastPct := float64(0)
	for lang, stat := range report {
		pct, _ := strconv.ParseFloat(stat.Percentage, 64)
		if pct > lastPct {
			language = lang
		}
		lastPct = pct
	}

	return
}

//
// Tag application
func tag(d *Data, language string) {
	_ = addon.Activity("adding language tag to the application")
	application, _ := addon.Application.Get(d.Application)

	//
	// Find or create tag type named 'Language'
	tagType := &model.TagType{}
	tagTypes, _ := addon.TagType.List()
	for _, tt := range tagTypes {
		if tt.Name == "Language" {
			tagType = &tt
		}
	}
	if tagType == nil {
		tagType.Name = "Language"
		_ = addon.TagType.Create(tagType)
	}

	//
	// Find or create tag named after the application language
	tag := &model.Tag{}
	tags, _ := addon.Tag.List()
	for _, t := range tags {
		if t.Name == language {
			tag = &t
		}
	}
	if tag == nil {
		_ = addon.Tag.Create(tag)
	}

	//
	// Append tag the application tags list.
	application.Tags = append(application.Tags, strconv.Itoa(int(tag.ID)))

	//
	// Update application.
	_ = addon.Application.Update(application)
}
