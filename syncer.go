package main

import (
	"fmt"
	"log"

	"github.com/google/go-github/github"
	"github.com/hashicorp/go-multierror"
	"github.com/xoebus/go-tracker"
)

var publicReposFilter = github.RepositoryListByOrgOptions{Type: "public"}
var openIssuesFilter = github.IssueListByRepoOptions{State: "open"}

type Syncer struct {
	GithubClient  *github.Client
	ProjectClient tracker.ProjectClient

	OrganizationName string

	RemainingGithubRequests int

	cachedUser *github.User
}

func (syncer *Syncer) SyncIssuesAndStories() error {
	repos, res, err := syncer.GithubClient.Repositories.ListByOrg(
		syncer.OrganizationName,
		&publicReposFilter,
	)
	if err != nil {
		return fmt.Errorf("failed to fetch issues: %s", err)
	}

	syncer.RemainingGithubRequests = res.Remaining

	var multiErr *multierror.Error

	for _, repo := range repos {
		err := syncer.processRepoIssues(repo)
		if err != nil {
			multiErr = multierror.Append(
				multiErr,
				fmt.Errorf("errors when processing %s/%s: %s", syncer.OrganizationName, *repo.Name, err),
			)
		}
	}

	return multiErr.ErrorOrNil()
}

func (syncer *Syncer) processRepoIssues(repo github.Repository) error {
	issues, _, err := syncer.GithubClient.Issues.ListByRepo(
		syncer.OrganizationName,
		*repo.Name,
		&openIssuesFilter,
	)
	if err != nil {
		return fmt.Errorf("failed to fetch issues for %s: %s", *repo.Name, err)
	}

	var multiErr *multierror.Error

	for _, issue := range issues {
		label := trackerLabelForIssue(repo, issue)

		err := syncer.ensureStoryExistsForIssue(repo, issue, label)
		if err != nil {
			multiErr = multierror.Append(
				multiErr,
				fmt.Errorf("failed to create story for issue %s: %s", label, err),
			)
		}
	}

	return multiErr.ErrorOrNil()
}

func (syncer *Syncer) ensureStoryExistsForIssue(
	repo github.Repository,
	issue github.Issue,
	label string,
) error {
	var allStories []tracker.Story

	query := tracker.StoriesQuery{
		Label: label,
	}

	for {
		stories, _, err := syncer.ProjectClient.Stories(query)
		if err != nil {
			return fmt.Errorf("failed to search for stories %s: %s", label, err)
		}

		if len(stories) == 0 {
			break
		}

		allStories = append(allStories, stories...)

		query.Offset = len(allStories)
	}

	if len(allStories) == 0 {
		story := storyForIssue(label, issue)

		createdStory, err := syncer.ProjectClient.CreateStory(story)
		if err != nil {
			return fmt.Errorf("failed to create story for %s: %s", label, err)
		}

		log.Println("created story for", label, "at", createdStory.URL)

		allStories = append(allStories, createdStory)
	}

	return syncer.ensureCommentWithStories(repo, issue, allStories)
}

func (syncer *Syncer) ensureCommentWithStories(
	repo github.Repository,
	issue github.Issue,
	allStories []tracker.Story,
) error {
	comments, err := syncer.allCommentsForIssue(repo, issue)
	if err != nil {
		return fmt.Errorf("failed to fetch issue comments: %s", err)
	}

	currentUser, err := syncer.currentUser()
	if err != nil {
		return fmt.Errorf("failed to get current user: %s", err)
	}

	var existingComment *github.IssueComment
	for _, comment := range comments {
		if *comment.User.ID == *currentUser.ID {
			existingComment = &comment
			break
		}
	}

	commentBody := `Hi there! We use Pivotal Tracker to provide visibility into what our team is working on.

We are keeping track of this issue in the following stories:

`

	for _, story := range allStories {
		check := " "
		if story.State == "accepted" {
			check = "x"
		}

		storyListEntry := fmt.Sprintf(
			"* [%s] [#%d](%s): %s",
			check,
			story.ID,
			story.URL,
			story.Name,
		)

		commentBody += storyListEntry + "\n"
	}

	commentBody += `
This comment will be automatically updated as the status in Tracker changes.

I will automatically close this issue when the above stories are done.`

	if existingComment == nil {
		createdComment, _, err := syncer.GithubClient.Issues.CreateComment(
			*repo.Owner.Login,
			*repo.Name,
			*issue.Number,
			&github.IssueComment{Body: &commentBody},
		)
		if err != nil {
			return fmt.Errorf("failed to create comment: %s", err)
		}

		log.Println("created comment:", *createdComment.HTMLURL)
	} else if *existingComment.Body != commentBody {
		existingComment.Body = &commentBody

		updatedComment, _, err := syncer.GithubClient.Issues.EditComment(
			*repo.Owner.Login,
			*repo.Name,
			*existingComment.ID,
			&github.IssueComment{Body: &commentBody},
		)
		if err != nil {
			return fmt.Errorf("failed to update comment: %s", err)
		}

		log.Println("updated comment:", *updatedComment.HTMLURL)
	}

	return nil
}

func (syncer *Syncer) allCommentsForIssue(repo github.Repository, issue github.Issue) ([]github.IssueComment, error) {
	options := &github.IssueListCommentsOptions{}

	var allComments []github.IssueComment

	for {
		comments, _, err := syncer.GithubClient.Issues.ListComments(
			*repo.Owner.Login,
			*repo.Name,
			*issue.Number,
			options,
		)
		if err != nil {
			return nil, err
		}

		if len(comments) == 0 {
			break
		}

		allComments = append(allComments, comments...)

		options.Page++
	}

	return allComments, nil
}

func (syncer *Syncer) currentUser() (*github.User, error) {
	if syncer.cachedUser == nil {
		user, _, err := syncer.GithubClient.Users.Get("")
		if err != nil {
			return nil, err
		}

		syncer.cachedUser = user
	}

	return syncer.cachedUser, nil
}

func trackerLabelForIssue(repo github.Repository, issue github.Issue) string {
	return fmt.Sprintf("%s/%s#%d", *repo.Owner.Login, *repo.Name, *issue.Number)
}

func storyForIssue(label string, issue github.Issue) tracker.Story {
	labels := []tracker.Label{
		{Name: label},
	}

	if issue.PullRequestLinks != nil {
		labels = append(labels, tracker.Label{Name: "has-pr"})
	}

	description := fmt.Sprintf(
		"[@%s](%s) opened [%s](%s) on %s:\n\n%s",
		*issue.User.Login,
		*issue.User.HTMLURL,
		label,
		*issue.HTMLURL,
		issue.CreatedAt.Format("January 2"),
		*issue.Body,
	)

	return tracker.Story{
		Name:        *issue.Title,
		Description: description,
		Type:        "feature",
		State:       "unscheduled",
		Labels:      labels,
	}
}
