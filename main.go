package main

import (
	"flag"
	"os"
	"reflect"

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

func required(thing interface{}, flag string) {
	if reflect.DeepEqual(thing, reflect.Zero(reflect.TypeOf(thing)).Interface()) {
		println("must specify " + flag)
		os.Exit(1)
	}
}

func main() {
	flag.Parse()

	required(*trackerToken, "--tracker-token")
	required(*githubToken, "--github-token")
	required(*projectID, "--project-id")
	required(*organizationName, "--organization")

	ghToken := &oauth2.Token{AccessToken: *githubToken}

	ghAuth := oauth2.NewClient(oauth2.NoContext, oauth2.StaticTokenSource(ghToken))

	githubClient := github.NewClient(ghAuth)

	trackerClient := tracker.NewClient(*trackerToken)
	projectClient := trackerClient.InProject(*projectID)

	syncer := &Syncer{
		GithubClient:  githubClient,
		ProjectClient: projectClient,

		OrganizationName: *organizationName,
	}

	syncer.BeAJerkToTrackerAPI()
}
