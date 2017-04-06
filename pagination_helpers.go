package main

import (
	"context"

	"github.com/google/go-github/github"
)

var publicReposFilter = github.RepositoryListByOrgOptions{Type: "public"}
var userPurblicReposFilter = github.RepositoryListOptions{Type: "public"}
var openIssuesFilter = github.IssueListByRepoOptions{State: "open"}

func (syncer *Syncer) reposToSync() ([]*github.Repository, error) {
	options := publicReposFilter

	var repos []*github.Repository

	// TODO: is there a more elgant way to handle checking by eithe org or by user without having to spin through two for loops?
	for {
		resources, resp, err := syncer.GithubClient.Repositories.ListByOrg(
			context.TODO(),
			syncer.OrganizationName,
			&options,
		)
		if err != nil {
			//return nil, err
		}

		if len(resources) == 0 {
			break
		}

		for _, repo := range resources {
			if syncer.shouldSync(repo) {
				repos = append(repos, repo)
			}
		}

		if resp.NextPage == 0 {
			break
		}

		options.ListOptions.Page = resp.NextPage
	}

	// if searching by org did not work, try searching by username
	if len(repos) == 0 {
		options := userPurblicReposFilter

		for {
			resources, resp, err := syncer.GithubClient.Repositories.List(
				syncer.OrganizationName,
				&options,
			)
			if err != nil {
				return nil, err
			}

			if len(resources) == 0 {
				break
			}

			for _, repo := range resources {
				if syncer.Repositories.IsEmpty() || syncer.Repositories.Contains(*repo.Name) {
					repos = append(repos, repo)
				}
			}

			if resp.NextPage == 0 {
				break
			}

			options.ListOptions.Page = resp.NextPage
		}
	}

	return repos, nil
}

func (syncer *Syncer) shouldSync(repository *github.Repository) bool {
	if len(syncer.Repositories) == 0 {
		return true
	}

	for _, name := range syncer.Repositories {
		if name == *repository.Name {
			return true
		}
	}

	return false
}

func (syncer *Syncer) allIssues(repo *github.Repository) ([]*github.Issue, error) {
	options := openIssuesFilter

	var all []*github.Issue

	for {
		resources, resp, err := syncer.GithubClient.Issues.ListByRepo(
			context.TODO(),
			*repo.Owner.Login,
			*repo.Name,
			&options,
		)
		if err != nil {
			return nil, err
		}

		if len(resources) == 0 {
			break
		}

		all = append(all, resources...)

		if resp.NextPage == 0 {
			break
		}

		options.ListOptions.Page = resp.NextPage
	}

	return all, nil
}

func (syncer *Syncer) allCommentsForIssue(
	repo *github.Repository,
	issue *github.Issue,
) ([]*github.IssueComment, error) {
	options := &github.IssueListCommentsOptions{}

	var all []*github.IssueComment

	for {
		resources, resp, err := syncer.GithubClient.Issues.ListComments(
			context.TODO(),
			*repo.Owner.Login,
			*repo.Name,
			*issue.Number,
			options,
		)
		if err != nil {
			return nil, err
		}

		if len(resources) == 0 {
			break
		}

		all = append(all, resources...)

		if resp.NextPage == 0 {
			break
		}

		options.ListOptions.Page = resp.NextPage
	}

	return all, nil
}
