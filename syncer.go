package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"text/template"

	"github.com/google/go-github/github"
	"github.com/hashicorp/go-multierror"
	"github.com/xoebus/go-tracker"
)

var storyStateCommentTemplate = template.Must(
	template.New("story-state").Parse(
		`Hi there!

We use Pivotal Tracker to provide visibility into what our team is working on. A story for this issue has been automatically created.

The current status is as follows:

{{range .}}* [{{if eq .State "accepted"}}x{{else}} {{end}}] [#{{.ID}}]({{.URL}}) {{.Name}}
{{end}}

This comment, as well as the labels on the issue, will be automatically updated as the status in Tracker changes.`,
	),
)

var issueClosedCommentTemplate = template.Must(
	template.New("issue-closed").Parse(
		`Hello again!

All stories related to this issue have been accepted, so I'm going to automatically close this issue.

At the time of writing, the following stories have been accepted:

{{range .}}* [#{{.ID}}]({{.URL}}) {{.Name}}
{{end}}

If you feel there is still more to be done, or if you have any questions, leave a comment and we'll reopen if necessary!`),
)

type Syncer struct {
	GithubClient  *github.Client
	ProjectClient tracker.ProjectClient

	OrganizationName string
	Repositories     StringSet

	cachedUser *github.User
}

func (syncer *Syncer) SyncIssuesAndStories() error {
	issues, repos, err := syncer.issuesAndReposToSync()
	if err != nil {
		return fmt.Errorf("failed to fetch issues: %s", err)
	}

	var multiErr *multierror.Error

	for _, repo := range repos {
		repoName := *repo.Owner.Login + "/" + *repo.Name

		log.Println("syncing labels for", repoName)

		if err := syncer.syncRepoStockLabels(repo); err != nil {
			log.Printf("failed setting up labels for %s: %s\n", repoName, err)
		}
	}

	for _, issue := range issues {
		label := trackerLabelForIssue(issue)

		err := syncer.ensureStoryExistsForIssue(issue, label)
		if err != nil {
			multiErr = multierror.Append(
				multiErr,
				fmt.Errorf("failed to create story for issue %s: %s", label, err),
			)
		}
	}

	return multiErr.ErrorOrNil()
}

func (syncer *Syncer) issuesAndReposToSync() ([]github.Issue, []github.Repository, error) {
	var issues []github.Issue
	var repos []github.Repository

	options := &github.SearchOptions{}

	query := syncer.issuesQuery()

	log.Println("syncing issues via query:", query)

	for {
		resources, resp, err := syncer.GithubClient.Search.Issues(query, nil)
		if err != nil {
			return nil, nil, err
		}

		if len(resources.Issues) == 0 {
			break
		}

		issues = append(issues, resources.Issues...)

		if resp.NextPage == 0 {
			break
		}

		options.ListOptions.Page = resp.NextPage
	}

	repoMap := map[int]github.Repository{}

	for _, issue := range issues {
		repoMap[*issue.Repository.ID] = *issue.Repository
	}

	for _, repo := range repoMap {
		repos = append(repos, repo)
	}

	return issues, repos, nil
}

func (syncer *Syncer) issuesQuery() string {
	queries := []string{"state:open"}

	if syncer.Repositories.IsEmpty() {
		queries = append(queries, "user:"+syncer.OrganizationName)
	} else {
		for _, repo := range syncer.Repositories.Members() {
			queries = append(queries, "repo:"+syncer.OrganizationName+"/"+repo)
		}
	}

	return strings.Join(queries, " ")
}

func (syncer *Syncer) syncRepoStockLabels(repo github.Repository) error {
	logName := *repo.Owner.Login + "/" + *repo.Name

	existingLabels, _, err := syncer.GithubClient.Issues.ListLabels(
		*repo.Owner.Login,
		*repo.Name,
		&github.ListOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to list labels for %s: %s", logName, err)
	}

	missingLabels := map[string]string{}
	for label, color := range storyStateLabels {
		missingLabels[label] = color
	}

	for label, color := range issueOnlyLabels {
		missingLabels[label] = color
	}

	for _, label := range existingLabels {
		color, found := missingLabels[*label.Name]
		if !found {
			continue
		}

		delete(missingLabels, *label.Name)

		if len(color) == 0 {
			// respect existing color
			continue
		}

		if color == *label.Color {
			// color already in sync; skip
			continue
		}

		log.Printf("updating label '%s' in repo %s\n", *label.Name, logName)

		_, _, err := syncer.GithubClient.Issues.EditLabel(
			*repo.Owner.Login,
			*repo.Name,
			*label.Name,
			&github.Label{
				Name:  label.Name,
				Color: &color,
			},
		)
		if err != nil {
			return fmt.Errorf("failed to update label '%s' in %s: %s", *label.Name, logName, err)
		}
	}

	for name, color := range missingLabels {
		log.Printf("creating label '%s' in repo %s\n", name, logName)

		_, _, err := syncer.GithubClient.Issues.CreateLabel(
			*repo.Owner.Login,
			*repo.Name,
			&github.Label{
				Name:  &name,
				Color: &color,
			},
		)
		if err != nil {
			return fmt.Errorf("failed to create label '%s' in %s: %s", name, logName, err)
		}
	}

	return nil
}

func (syncer *Syncer) ensureStoryExistsForIssue(issue github.Issue, label string) error {
	log.Printf("syncing %s: %s\n", label, *issue.Title)

	var allStories StorySet

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
		// no stories for the issue yet; create an initial one

		story := choreForNewIssue(label, issue)

		createdStory, err := syncer.ProjectClient.CreateStory(story)
		if err != nil {
			return fmt.Errorf("failed to create story for %s: %s", label, err)
		}

		log.Println("created story for", label, "at", createdStory.URL)

		allStories = append(allStories, createdStory)

	} else if allStories.AllAccepted() && issue.UpdatedAt.After(allStories.LastAccepted()) {
		// issue has been reopened

		story := choreForReopenedIssue(label, issue)

		createdStory, err := syncer.ProjectClient.CreateStory(story)
		if err != nil {
			return fmt.Errorf("failed to create story for %s: %s", label, err)
		}

		log.Println("created chore for reopening of", label, "at", createdStory.URL)

		allStories = append(allStories, createdStory)
	}

	if len(allStories) == 1 && (allStories.Untriaged() || allStories.Unscheduled()) {
		story := allStories[0]

		syncedStory, err := syncer.syncStoryFromIssue(story, issue)
		if err != nil {
			return fmt.Errorf("failed to sync story type for %d: %s", story.ID, err)
		}

		allStories[0] = syncedStory
	}

	if issue.PullRequestLinks != nil && !allStories.HasPR() {
		if err := syncer.setHasPR(allStories); err != nil {
			return fmt.Errorf("failed to set has-pr label for stories: %s", err)
		}
	} else if issue.PullRequestLinks == nil && allStories.HasPR() {
		if err := syncer.unsetHasPR(allStories); err != nil {
			return fmt.Errorf("failed to remove has-pr label for stories: %s", err)
		}
	}

	if err := syncer.ensureCommentWithStories(issue, allStories); err != nil {
		return fmt.Errorf("failed to upsert comment for stories: %s", err)
	}

	if err := syncer.syncIssueLabels(issue, allStories.IssueLabels()); err != nil {
		return fmt.Errorf("failed to sync story labels: %s", err)
	}

	if allStories.AllAccepted() {
		log.Println("all stories for", label, "are accepted; closing!")

		err := syncer.closeIssue(issue, allStories)
		if err != nil {
			return fmt.Errorf("failed to close issue: %s", err)
		}
	}

	return nil
}

func (syncer *Syncer) setHasPR(stories StorySet) error {
	for _, story := range stories {
		if (StorySet{story}).HasPR() {
			continue
		}

		log.Printf("adding has-pr label to #%d\n", story.ID)

		_, err := syncer.ProjectClient.AddStoryLabel(story.ID, "has-pr")
		if err != nil {
			return err
		}
	}

	return nil
}

func (syncer *Syncer) unsetHasPR(stories StorySet) error {
	for _, story := range stories {
		if !(StorySet{story}).HasPR() {
			continue
		}

		for _, label := range story.Labels {
			if label.Name == "has-pr" {
				log.Printf("removing has-pr label from #%d\n", story.ID)

				err := syncer.ProjectClient.RemoveStoryLabel(story.ID, label.ID)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (syncer *Syncer) ensureCommentWithStories(issue github.Issue, allStories []tracker.Story) error {
	comments, err := syncer.allCommentsForIssue(issue)
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
			existingComment = comment
			break
		}
	}

	buf := new(bytes.Buffer)
	if err := storyStateCommentTemplate.Execute(buf, allStories); err != nil {
		return fmt.Errorf("error building comment body: %s", err)
	}

	commentBody := buf.String()

	if existingComment == nil {
		createdComment, _, err := syncer.GithubClient.Issues.CreateComment(
			*issue.Repository.Owner.Login,
			*issue.Repository.Name,
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
			*issue.Repository.Owner.Login,
			*issue.Repository.Name,
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

func (syncer *Syncer) allCommentsForIssue(issue github.Issue) ([]*github.IssueComment, error) {
	options := &github.IssueListCommentsOptions{}

	var all []*github.IssueComment

	for {
		resources, resp, err := syncer.GithubClient.Issues.ListComments(
			*issue.Repository.Owner.Login,
			*issue.Repository.Name,
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

func (syncer *Syncer) syncIssueLabels(issue github.Issue, labels []string) error {
	existingLabels := map[string]bool{}
	for _, label := range issue.Labels {
		existingLabels[*label.Name] = true
	}

	labelsToAdd := []string{}
	for _, label := range labels {
		if !existingLabels[label] {
			labelsToAdd = append(labelsToAdd, label)
		}
	}

	labelsToRemove := []string{}
	for stockLabel, _ := range storyStateLabels {
		if !existingLabels[stockLabel] {
			continue
		}

		stillHasLabel := false
		for _, label := range labels {
			if label == stockLabel {
				stillHasLabel = true
				break
			}
		}

		if !stillHasLabel {
			labelsToRemove = append(labelsToRemove, stockLabel)
		}
	}

	if len(labelsToRemove) == 0 && len(labelsToAdd) == 0 {
		return nil
	}

	log.Println("setting issue labels:", strings.Join(labels, ", "))

	for _, label := range labelsToRemove {
		_, err := syncer.GithubClient.Issues.RemoveLabelForIssue(
			*issue.Repository.Owner.Login,
			*issue.Repository.Name,
			*issue.Number,
			label,
		)
		if err != nil && !strings.Contains(err.Error(), "404") {
			return fmt.Errorf("failed to remove label '%s': %s", label, err)
		}
	}

	_, _, err := syncer.GithubClient.Issues.AddLabelsToIssue(
		*issue.Repository.Owner.Login,
		*issue.Repository.Name,
		*issue.Number,
		labelsToAdd,
	)
	if err != nil {
		return fmt.Errorf("failed to add labels to issue: %s", err)
	}

	return nil
}

func (syncer *Syncer) closeIssue(issue github.Issue, stories StorySet) error {
	buf := new(bytes.Buffer)
	if err := issueClosedCommentTemplate.Execute(buf, stories); err != nil {
		return fmt.Errorf("error building comment body: %s", err)
	}

	closedMessage := buf.String()

	_, _, err := syncer.GithubClient.Issues.CreateComment(
		*issue.Repository.Owner.Login,
		*issue.Repository.Name,
		*issue.Number,
		&github.IssueComment{Body: &closedMessage},
	)
	if err != nil {
		return fmt.Errorf("failed to leave closed message: %s", err)
	}

	state := "closed"
	_, _, err = syncer.GithubClient.Issues.Edit(
		*issue.Repository.Owner.Login,
		*issue.Repository.Name,
		*issue.Number,
		&github.IssueRequest{State: &state},
	)
	if err != nil {
		return fmt.Errorf("failed to close issue: %s", err)
	}

	return err
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

func (syncer *Syncer) syncStoryFromIssue(story tracker.Story, issue github.Issue) (tracker.Story, error) {
	storyType := issueStoryType(issue)

	var err error

	if story.State == tracker.StoryStateStarted && story.Type == tracker.StoryTypeChore && storyType != tracker.StoryTypeChore {
		log.Println("moving story to icebox...")

		story, err = syncer.ProjectClient.UnscheduleStory(story.ID)
		if err != nil {
			return tracker.Story{}, err
		}
	}

	if story.Type != storyType {
		log.Printf("updating story type to '%s'...\n", storyType)
		story, err = syncer.ProjectClient.SetStoryType(story.ID, storyType)
		if err != nil {
			return tracker.Story{}, err
		}
	}

	if story.Name != *issue.Title {
		log.Println("syncing story name...")
		story, err = syncer.ProjectClient.SetStoryName(story.ID, *issue.Title)
		if err != nil {
			return tracker.Story{}, err
		}
	}

	return story, nil
}

func trackerLabelForIssue(issue github.Issue) string {
	return fmt.Sprintf("%s/%s#%d", *issue.Repository.Owner.Login, *issue.Repository.Name, *issue.Number)
}

func choreForNewIssue(label string, issue github.Issue) tracker.Story {
	labels := []tracker.Label{
		{Name: label},
	}

	if issue.PullRequestLinks != nil {
		prJSON, _ := json.Marshal(issue.PullRequestLinks)
		log.Printf("  has pull request: %s\n", string(prJSON))
		labels = append(labels, tracker.Label{Name: "has-pr"})
	}

	description := fmt.Sprintf(
		"[@%s](%s) opened [%s](%s) on %s",
		*issue.User.Login,
		*issue.User.HTMLURL,
		label,
		*issue.HTMLURL,
		issue.CreatedAt.Format("January 2"),
	)

	return tracker.Story{
		Name:        *issue.Title,
		Description: description,
		Type:        "chore",
		State:       "unscheduled",
		Labels:      labels,
	}
}

func choreForReopenedIssue(label string, issue github.Issue) tracker.Story {
	labels := []tracker.Label{
		{Name: label},
	}

	description := fmt.Sprintf(
		"[@%s](%s) reopened [%s](%s) on %s",
		*issue.User.Login,
		*issue.User.HTMLURL,
		label,
		*issue.HTMLURL,
		issue.UpdatedAt.Format("January 2"),
	)

	return tracker.Story{
		Name:        "reopened: " + *issue.Title,
		Description: description,
		Type:        "chore",
		State:       "unscheduled",
		Labels:      labels,
	}
}

func issueStoryType(issue github.Issue) tracker.StoryType {
	if issueHasLabel(issue, IssueLabelEnhancement) {
		return tracker.StoryTypeFeature
	}

	if issueHasLabel(issue, IssueLabelBug) {
		return tracker.StoryTypeBug
	}

	return tracker.StoryTypeChore
}

func issueHasLabel(issue github.Issue, needle string) bool {
	for _, label := range issue.Labels {
		if *label.Name == needle {
			return true
		}
	}

	return false
}
