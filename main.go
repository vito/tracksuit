package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"reflect"

	flags "github.com/jessevdk/go-flags"

	"github.com/google/go-github/github"
	"github.com/vito/twentythousandtonnesofcrudeoil"
	"github.com/xoebus/go-tracker"
	"golang.org/x/oauth2"
)

type TracksuitCommand struct {
	GitHub struct {
		Token            string   `long:"token"	required:"true" description:"GitHub access token"`
		OrganizationName string   `long:"organization-name" required:"true" description:"GitHub organization name"`
		Repositories     []string `long:"repository" desciption:"Repository to sync. Can be repeated to sync many repositories. If omitted, all repositories are synced."`
		ApiUrl           string   `long:"api-url" description:"Github api url. If omitted it defaults to api.github.com"`
	} `group:"GitHub Configuration" namespace:"github"`

	Tracker struct {
		Token     string `long:"token"			required:"true" description:"Tracker Access token"`
		ProjectID int    `long:"project-id" required:"true" description:"Tracker project ID"`
	} `group:"Pivotal Tracker Configuration" namespace:"tracker"`

	AdditionalLabels map[string]string `long:"label" value-name:"NAME:COLOR" description:"Additional labels to sync up between GitHub and Tracker. They will be created on the synced GitHub repositories automatically."`

	GCLabels bool `long:"gc-labels" description:"Garbage collect labels in Tracker that no longer reference an issue"`
}

var repositories = NewStringSet()

func required(thing interface{}, flag string) {
	if reflect.DeepEqual(thing, reflect.Zero(reflect.TypeOf(thing)).Interface()) {
		println("must specify " + flag)
		os.Exit(1)
	}
}

func (cmd *TracksuitCommand) Execute(argv []string) error {
	ghToken := &oauth2.Token{AccessToken: cmd.GitHub.Token}

	ghAuth := oauth2.NewClient(oauth2.NoContext, oauth2.StaticTokenSource(ghToken))

	githubClient := github.NewClient(ghAuth)

	if cmd.GitHub.ApiUrl != "" {
		var Url *url.URL
		Url, _ = url.Parse(cmd.GitHub.ApiUrl)
		githubClient.BaseURL = Url
	}

	trackerClient := tracker.NewClient(cmd.Tracker.Token)
	projectClient := trackerClient.InProject(cmd.Tracker.ProjectID)

	syncer := &Syncer{
		GithubClient:  githubClient,
		ProjectClient: projectClient,

		OrganizationName: cmd.GitHub.OrganizationName,
		Repositories:     cmd.GitHub.Repositories,

		AdditionalLabels: cmd.AdditionalLabels,
	}

	if err := syncer.SyncIssuesAndStories(); err != nil {
		return err
	}

	log.Println("synced")

	if cmd.GCLabels {
		log.Println("gcing labels")

		gcer := &LabelGCer{
			ProjectClient: projectClient,
		}

		gcer.GC()
	}

	return nil
}

func main() {
	cmd := &TracksuitCommand{}

	parser := flags.NewParser(cmd, flags.Default)
	parser.NamespaceDelimiter = "-"

	twentythousandtonnesofcrudeoil.TheEnvironmentIsPerfectlySafe(parser, "TRACKSUIT_")

	args, err := parser.Parse()
	if err != nil {
		os.Exit(1)
	}

	err = cmd.Execute(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
