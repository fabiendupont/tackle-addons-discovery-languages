package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	hub "github.com/konveyor/tackle-hub/addon"
	"github.com/konveyor/tackle-hub/api"
	//	"github.com/konveyor/tackle-hub/model"
	"os"
	"os/exec"
	pathlib "path"
	"strconv"
)

const (
	Kind          = "File:application/json"
	LanguagesFile = "/tmp/languages.json"
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
	var err error
	fmt.Printf("--- Tackle Addon - Discovery - Languages ---\n")

	// Get the addon data associated with the task.
	d := &Data{}
	_ = addon.DataWith(d)

	fmt.Printf("Data passed to the addon:\n")
	fmt.Printf("  - Application ID: %d\n", d.Application)
	fmt.Printf("  - Git URL: %s\n", d.GitURL)
	fmt.Printf("  - Git Branch: %s\n", d.GitBranch)
	fmt.Printf("  - Git Path: %s\n", d.GitPath)

	// Error handler
	defer func() {
		if err != nil {
			fmt.Printf("Addon failed: %s\n", err.Error())
			_ = addon.Failed(err.Error())
			os.Exit(1)
		}
	}()

	// Signal that addon has started
	fmt.Printf("Addon started\n")
	_ = addon.Started()

	// Validate the addon data and enforce defaults
	if d.Application == 0 {
		fmt.Printf("Field 'application' is missing in addon data\n")
		_ = addon.Failed("field 'application' is missing in addon data")
		os.Exit(1)
	}
	if d.GitURL == "" {
		fmt.Printf("Field 'git_url' is missing in addon data\n")
		_ = addon.Failed("field 'git_url' is missing in addon data")
		os.Exit(1)
	}

	// Clone Git repository
	cloneGitRepository(d)

	// Get the main language
	language, err := getLanguage(d)
	if err != nil {
		return
	}

	// Upload full languages list as an artifact
	err = createBucket(d)
	if err != nil {
		return
	}

	// Set language tag for the application
	err = tag(d, language)
	if err != nil {
		return
	}

	// Task update: The addon has succeeded
	_ = addon.Succeeded()
}

//
// Clone Git repository
// TODO: Add support for non anonymous Git operations
// TODO: Add support for fetching credentials from Hub
func cloneGitRepository(d *Data) (err error) {
	fmt.Printf("Cloning Git repository\n")
	_ = addon.Activity("cloning Git repository")

	gitCloneOptions := &git.CloneOptions{
		URL:               d.GitURL,
		ReferenceName:     plumbing.ReferenceName(d.GitBranch),
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	}
	fmt.Printf("Git clone options\n")
	fmt.Printf("  - URL: %s\n", gitCloneOptions.URL)
	fmt.Printf("  - ReferenceName: %s\n", gitCloneOptions.ReferenceName)
	//fmt.Printf("  - RecurseSubmodules: %s", git.DefaultSubmoduleRecursionDepth)

	_, err = git.PlainClone("/tmp/app", false, gitCloneOptions)
	if err != nil {
		return
	}

	fmt.Printf("Git clone completed\n")

	return
}

//
// Returns the most represented language in the repository
func getLanguage(d *Data) (language string, err error) {
	var cmd *exec.Cmd
	fmt.Printf("Identifying languages in the repository\n")
	_ = addon.Activity("identifying languages in the repository")

	appPath := "/tmp/app/" + d.GitPath
	fmt.Printf("Application path to analyze: %s\n", appPath)

	// GitHub Linguist only supports running at the root of
	// the Git repository. When we need to analyze only a
	// subfolder, we have to initialize it as a Git repository
	// and run github-linguist in this new repository.
	if d.GitPath != "" {
		fmt.Printf("Initialize %s as a Git repository\n", appPath)
		cmd = exec.Command("/usr/bin/git", "init")
		cmd.Dir = appPath
		err = cmd.Run()
		if err != nil {
			return
		}

		fmt.Printf("Configure the user email locally\n")
		cmd = exec.Command("/usr/bin/git", "config", "user.email", "foo@bar")
		cmd.Dir = appPath
		err = cmd.Run()
		if err != nil {
			return
		}

		fmt.Printf("Add all files in %s to the stage\n", appPath)
		cmd = exec.Command("/usr/bin/git", "add", ".")
		cmd.Dir = appPath
		err = cmd.Run()
		if err != nil {
			return
		}

		fmt.Printf("Create a commit to have a Git index\n")
		cmd = exec.Command("/usr/bin/git", "commit", "-m", "GitHub Linguist")
		cmd.Dir = appPath
		err = cmd.Run()
		if err != nil {
			return
		}
	}

	cmd = exec.Command("/usr/local/bin/github-linguist", "--json", appPath)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	fmt.Printf("Calling GitHub Linguist...\n")
	err = cmd.Run()
	if err != nil {
		return
	}
	fmt.Printf("Analysis completed\n")

	// Write the JSON output to a file
	// TODO: Allow upload of content, not only path
	fmt.Printf("Storing the resulting JSON output in %s\n", LanguagesFile)
	f, err := os.Create(LanguagesFile)
	if err != nil {
		return
	}
	_, err = f.WriteString(stdout.String())
	if err != nil {
		return
	}
	f.Close()

	fmt.Printf("Sorting the languages by percentage\n")
	// Read the JSON output into a map
	type statistic struct {
		Size       string `json:"size"`
		Percentage string `json:"percentage"`
	}
	report := map[string]statistic{}
	_ = json.Unmarshal([]byte(stdout.String()), &report)
	highestPct := float64(0)
	for lang, stat := range report {
		pct, _ := strconv.ParseFloat(stat.Percentage, 64)
		fmt.Printf("Language '%s' has percentage '%.2f'\n", lang, pct)
		if pct > highestPct {
			fmt.Printf("%.2f > %.2f - Highest language is %s\n", pct, highestPct, lang)
			language = lang
			highestPct = pct
		}
	}
	fmt.Printf("Most represented language is: %s\n", language)

	return
}

//
// Upload full languages list as an artifact
func createBucket(d *Data) (err error) {
	fmt.Printf("Storing the full result into a bucket\n")
	_ = addon.Activity("storing the full result into a bucket\n")
	_ = addon.Total(1)

	fmt.Printf("Creating a bucket\n")
	bucket := &api.Bucket{}
	bucket.CreateUser = "addon"
	bucket.Name = "DiscoveryLanguage"
	bucket.ApplicationID = d.Application
	err = addon.Bucket.Create(bucket)
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			_ = addon.Bucket.Delete(bucket)
		}
	}()

	var b []byte
	b, err = os.ReadFile(LanguagesFile)
	if err != nil {
		return
	}
	target := pathlib.Join(bucket.Path, pathlib.Base(LanguagesFile))
	fmt.Printf("Copying %s to %s\n", LanguagesFile, target)
	err = os.WriteFile(target, b, 0644)
	if err != nil {
		return
	}
	_ = addon.Increment()

	return
}

//
// Tag application
func tag(d *Data, language string) (err error) {
	fmt.Printf("Adding the language tag to the application\n")
	_ = addon.Activity("adding language tag to the application")
	application, _ := addon.Application.Get(d.Application)
	fmt.Printf("Application id is %d\n", application.ID)

	//
	// Find or create tag type named 'Language'
	tagType := &api.TagType{}
	tagTypes, _ := addon.TagType.List()
	for _, tt := range tagTypes {
		if tt.Name == "Language" {
			fmt.Printf("Found tag type 'Language' with id %d\n", tt.ID)
			tagType = &tt
			break
		}
	}
	if tagType == nil {
		fmt.Printf("Tag type 'Language' does not exist. Creating it.\n")
		tagType.Name = "Language"
		_ = addon.TagType.Create(tagType)
	}
	fmt.Printf("Tag type 'Language' has id %d\n", tagType.ID)

	//
	// Find or create tag named after the application language
	var tag *api.Tag
	tags, _ := addon.Tag.List()
	for _, t := range tags {
		if t.TagType.ID == tagType.ID && t.Name == language {
			fmt.Printf("Found tag 'Language/%s with id %d\n", language, t.ID)
			tag = &t
			break
		}
	}
	if tag == nil {
		fmt.Printf("Tag 'Language/%s' does not exist. Creating it.\n", language)
		tag = &api.Tag{Name: language}
		tag.TagType.ID = tagType.ID
		_ = addon.Tag.Create(tag)
	}
	fmt.Printf("Tag 'Language/%s' has id %d\n", language, tag.ID)

	//
	// Append tag the application tags list.
	fmt.Printf("Add tag 'Language/%s' to application '%s'\n", language, application.Name)
	application.Tags = append(application.Tags, strconv.Itoa(int(tag.ID)))

	//
	// Update application.
	fmt.Printf("Saving the tag information\n")
	err = addon.Application.Update(application)

	return
}
