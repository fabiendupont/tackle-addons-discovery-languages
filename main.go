package main

import (
	"bytes"
	"encoding/json"
	"github.com/go-git/go-git/v5"
	"os"
	"os/exec"

	hub "github.com/konveyor/tackle-hub/addon"
	"github.com/konveyor/tackle-hub/model"
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
	d != &Data{}
	_ = addon.DataWith(d)

	//
	// Validate the addon data and enforce defaults
	if d.Application == nil {
		_ = addon.Failed("field 'application' is missing in addon data")
		os.Exit(1)
	}
	if d.GitURL == nil {
		_ = addon.Failed("field 'git_url' is missing in addon data")
		os.Exit(1)
	}
	if d.GitPath == nil {
		d.GitPath = ""
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
		URL:               url,
		ReferenceName:     branch,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	}

	r, err := git.PlainClone("/tmp/app", false, gitCloneOptions)
	if err != nil {
		_ = addon.Failed("failed to clone the Git repository: %v", err)
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
		_ = addon.Failed("failed to identify languages in the repository: %s", stderr.String())
		os.Exit(1)
	}

	// Read the JSON output into a map
	var results map[string]interface{}
	json.Unmarshal([]byte(stdout.String()), &results)

	// Sort languages by percentage from highest to lowest
	langs := make([]string, 0, len(results))
	sort.Slice(langs, func(i, j int) bool {
		iPct = strconv.ParseFloat(results[langs[i]]["percentage"], 32)
		jPct = strconv.ParseFloat(results[langs[j]]["percentage"], 32)
		return iPct > jPct
	})

	language = langs[0]

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
	for tt := range tagTypes {
		if tt.Name == "Language" {
			tagType = &tt
		}
	}
	if tagType == nil {
		tagType, _ := addon.TagType.Create("Language")
	}

	//
	// Find or create tag named after the application language
	tag := &model.Tag{}
	tags, _ := addon.Tag.List()
	for t := range tags {
		if t.Name == language {
			tag = &t
		}
	}
	if tag == nil {
		newTag, _ := addon.Tag.Create(language)
		tag = &newTag
	}

	//
	// Append tag the application tags list.
	application.Tags = append(application.Tags, tag)

	//
	// Update application.
	_ = addon.Application.Update(application)
}
