package tracker

import (
	"fmt"
	"net/url"
	"strings"
)

type Query interface {
	Query() url.Values
}

type StoriesQuery struct {
	State  StoryState
	Label  string
	Filter []string

	Limit  int
	Offset int
}

func (query StoriesQuery) Query() url.Values {
	params := url.Values{}

	if query.State != "" {
		params.Set("with_state", string(query.State))
	}

	if query.Label != "" {
		params.Set("with_label", query.Label)
	}

	if len(query.Filter) != 0 {
		params.Set("filter", strings.Join(query.Filter, " "))
	}

	if query.Limit != 0 {
		params.Set("limit", fmt.Sprintf("%d", query.Limit))
	}

	if query.Offset != 0 {
		params.Set("offset", fmt.Sprintf("%d", query.Offset))
	}

	return params
}

type ActivityQuery struct {
	Limit          int
	Offset         int
	OccurredBefore int64
	OccurredAfter  int64
	SinceVersion   int
}

func (query ActivityQuery) Query() url.Values {
	params := url.Values{}

	if query.Limit != 0 {
		params.Set("limit", fmt.Sprintf("%d", query.Limit))
	}

	if query.Offset != 0 {
		params.Set("offset", fmt.Sprintf("%d", query.Offset))
	}

	if query.OccurredBefore != 0 {
		params.Set("occurred_before", fmt.Sprintf("%d", query.OccurredBefore))
	}

	if query.OccurredAfter != 0 {
		params.Set("occurred_after", fmt.Sprintf("%d", query.OccurredAfter))
	}

	if query.SinceVersion != 0 {
		params.Set("since_version", fmt.Sprintf("%d", query.SinceVersion))
	}

	return params
}
