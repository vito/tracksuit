package tracker

import "time"

type Me struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Initials string `json:"initials"`
	ID       int    `json:"id"`
	Email    string `json:"email"`
}

type Project struct {
	Id int
}

type Story struct {
	ID        int `json:"id,omitempty"`
	ProjectID int `json:"project_id,omitempty"`

	URL string `json:"url,omitempty"`

	Name        string     `json:"name,omitempty"`
	Description string     `json:"description,omitempty"`
	Type        StoryType  `json:"story_type,omitempty"`
	State       StoryState `json:"current_state,omitempty"`

	Labels []Label `json:"labels,omitempty"`

	CreatedAt  *time.Time `json:"created_at,omitempty"`
	UpdatedAt  *time.Time `json:"updated_at,omitempty"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
}

type Comment struct {
	Text string `json:"text,omitempty"`
}

type Label struct {
	ID        int `json:"id,omitempty"`
	ProjectID int `json:"project_id,omitempty"`

	Name string `json:"name"`
}

type StoryType string

const (
	StoryTypeFeature = "feature"
	StoryTypeBug     = "bug"
	StoryTypeChore   = "chore"
	StoryTypeRelease = "release"
)

type StoryState string

const (
	StoryStateUnscheduled = "unscheduled"
	StoryStatePlanned     = "planned"
	StoryStateStarted     = "started"
	StoryStateFinished    = "finished"
	StoryStateDelivered   = "delivered"
	StoryStateAccepted    = "accepted"
	StoryStateRejected    = "rejected"
)

type Activity struct {
	Kind             string        `json:"kind"`
	GUID             string        `json:"guid"`
	ProjectVersion   int           `json:"project_version"`
	Message          string        `json:"message"`
	Highlight        string        `json:"highlight"`
	Changes          []interface{} `json:"changes"`
	PrimaryResources []interface{} `json:"primary_resources"`
	Project          interface{}   `json:"project"`
	PerformedBy      interface{}   `json:"performed_by"`
	OccurredAt       int64         `json:"occurred_at"`
}
