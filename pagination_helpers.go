package main

import "github.com/google/go-github/github"

func (syncer *Syncer) allRepos() ([]github.Repository, error) {
	options := publicReposFilter

	var all []github.Repository

	for {
		resources, _, err := syncer.GithubClient.Repositories.ListByOrg(
			syncer.OrganizationName,
			&options,
		)
		if err != nil {
			return nil, err
		}

		if len(resources) == 0 {
			break
		}

		all = append(all, resources...)

		options.Page++
	}

	return all, nil
}

func (syncer *Syncer) allIssues(repo github.Repository) ([]github.Issue, error) {
	options := openIssuesFilter

	var all []github.Issue

	for {
		resources, _, err := syncer.GithubClient.Issues.ListByRepo(
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

		options.Page++
	}

	return all, nil
}

func (syncer *Syncer) allCommentsForIssue(repo github.Repository, issue github.Issue) ([]github.IssueComment, error) {
	options := &github.IssueListCommentsOptions{}

	var all []github.IssueComment

	for {
		resources, _, err := syncer.GithubClient.Issues.ListComments(
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

		options.Page++
	}

	return all, nil
}
