package main

import (
	"flag"
	"log"
	"os"
	"reflect"
	"net/url"
	
	"github.com/google/go-github/github"
	"github.com/xoebus/go-tracker"
	"golang.org/x/oauth2"
)

var githubToken = flag.String(
	"github-token",
	"",
	"Github access token",
)

var trackerToken = flag.String(
	"tracker-token",
	"",
	"Pivotal Tracker access token",
)

var projectID = flag.Int(
	"project-id",
	0,
	"Tracker project ID",
)

var organizationName = flag.String(
	"organization",
	"",
	"Github organization name",
)

var githuburl = flag.String(
	"githuburl",
	"",
	"Github ENTERPRISE URL",
)


var repositories = NewStringSet()

func required(thing interface{}, flag string) {
	if reflect.DeepEqual(thing, reflect.Zero(reflect.TypeOf(thing)).Interface()) {
		println("must specify " + flag)
		os.Exit(1)
	}
}

func main() {
	flag.Var(
		&repositories,
		"repositories",
		"Comma separated list of repositories to sync, can be provided more than once (default: all in provided organization)",
	)

	flag.Parse()

	required(*trackerToken, "--tracker-token")
	required(*githubToken, "--github-token")
	required(*projectID, "--project-id")
	required(*organizationName, "--organization")

	ghToken := &oauth2.Token{AccessToken: *githubToken}

	ghAuth := oauth2.NewClient(oauth2.NoContext, oauth2.StaticTokenSource(ghToken))

	githubClient := github.NewClient(ghAuth)

	if *githuburl != "" {
		var Url *url.URL
		Url, err := url.Parse(*githuburl)
		if err != nil {
			panic(err)
		}
		githubClient.BaseURL = Url
	}

  trackerClient := tracker.NewClient(*trackerToken)
  projectClient := trackerClient.InProject(*projectID)

	syncer := &Syncer{
		GithubClient:  githubClient,
		ProjectClient: projectClient,

		OrganizationName: *organizationName,
		Repositories:     repositories,
	}

	if err := syncer.SyncIssuesAndStories(); err != nil {
		log.Println("failed to sync; remaining requests:", syncer.GithubClient.Rate().Remaining)
		println("")
		println(err.Error())
		os.Exit(1)
	}

	log.Println("synced; remaining requests:", syncer.GithubClient.Rate().Remaining)
}
