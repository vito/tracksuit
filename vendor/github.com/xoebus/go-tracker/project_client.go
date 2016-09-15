package tracker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

type ProjectClient struct {
	id   int
	conn connection
}

func (p ProjectClient) Stories(query StoriesQuery) ([]Story, Pagination, error) {
	params := query.Query().Encode()

	request, err := p.createRequest("GET", "/stories?"+params)
	if err != nil {
		return nil, Pagination{}, err
	}

	var stories []Story
	pagination, err := p.conn.Do(request, &stories)
	if err != nil {
		return nil, Pagination{}, err
	}

	return stories, pagination, err
}

func (p ProjectClient) Labels() ([]Label, error) {
	request, err := p.createRequest("GET", "/labels")
	if err != nil {
		return nil, err
	}

	var labels []Label
	_, err = p.conn.Do(request, &labels)
	if err != nil {
		return nil, err
	}

	return labels, err
}

func (p ProjectClient) StoryActivity(storyId int, query ActivityQuery) (activities []Activity, err error) {
	url := fmt.Sprintf("/stories/%d/activity", storyId)
	params := query.Query().Encode()
	request, err := p.createRequest("GET", url+"?"+params)
	if err != nil {
		return activities, err
	}

	_, err = p.conn.Do(request, &activities)
	return activities, err
}

func (p ProjectClient) DeliverStoryWithComment(storyId int, comment string) (Story, error) {
	story, err := p.DeliverStory(storyId)
	if err != nil {
		return Story{}, err
	}

	url := fmt.Sprintf("/stories/%d/comments", storyId)
	request, err := p.createRequest("POST", url)
	if err != nil {
		return Story{}, err
	}

	buffer := &bytes.Buffer{}
	json.NewEncoder(buffer).Encode(Comment{
		Text: comment,
	})

	p.addJSONBodyReader(request, buffer)

	_, err = p.conn.Do(request, nil)
	if err != nil {
		return Story{}, err
	}

	return story, nil
}

func (p ProjectClient) DeliverStory(storyId int) (Story, error) {
	url := fmt.Sprintf("/stories/%d", storyId)
	request, err := p.createRequest("PUT", url)
	if err != nil {
		return Story{}, err
	}

	p.addJSONBody(request, `{"current_state":"delivered"}`)

	var updatedStory Story
	_, err = p.conn.Do(request, &updatedStory)
	return updatedStory, err
}

func (p ProjectClient) CreateStory(story Story) (Story, error) {
	request, err := p.createRequest("POST", "/stories")
	if err != nil {
		return Story{}, err
	}

	buffer := &bytes.Buffer{}
	json.NewEncoder(buffer).Encode(story)

	p.addJSONBodyReader(request, buffer)

	var createdStory Story
	_, err = p.conn.Do(request, &createdStory)
	return createdStory, err
}

func (p ProjectClient) DeleteStory(storyId int) error {
	url := fmt.Sprintf("/stories/%d", storyId)
	request, err := p.createRequest("DELETE", url)
	if err != nil {
		return err
	}

	_, err = p.conn.Do(request, nil)
	return err
}

func (p ProjectClient) DeleteLabel(labelId int) error {
	url := fmt.Sprintf("/labels/%d", labelId)
	request, err := p.createRequest("DELETE", url)
	if err != nil {
		return err
	}

	_, err = p.conn.Do(request, nil)
	return err
}

func (p ProjectClient) AddStoryLabel(storyId int, label string) (Label, error) {
	url := fmt.Sprintf("/stories/%d/labels", storyId)
	request, err := p.createRequest("POST", url)
	if err != nil {
		return Label{}, err
	}

	reqJSON, err := json.Marshal(Label{Name: label})
	if err != nil {
		return Label{}, err
	}

	p.addJSONBody(request, string(reqJSON))

	var createdLabel Label
	_, err = p.conn.Do(request, &createdLabel)
	return createdLabel, err
}

func (p ProjectClient) RemoveStoryLabel(storyId int, labelId int) error {
	url := fmt.Sprintf("/stories/%d/labels/%d", storyId, labelId)
	request, err := p.createRequest("DELETE", url)
	if err != nil {
		return err
	}

	_, err = p.conn.Do(request, nil)
	return err
}

func (p ProjectClient) SetStoryType(storyId int, storyType StoryType) (Story, error) {
	url := fmt.Sprintf("/stories/%d", storyId)
	request, err := p.createRequest("PUT", url)
	if err != nil {
		return Story{}, err
	}

	p.addJSONBody(request, fmt.Sprintf(`{"story_type":%q}`, storyType))

	var updatedStory Story
	_, err = p.conn.Do(request, &updatedStory)
	return updatedStory, err
}

func (p ProjectClient) createRequest(method string, path string) (*http.Request, error) {
	projectPath := fmt.Sprintf("/projects/%d%s", p.id, path)
	return p.conn.CreateRequest(method, projectPath)
}

func (p ProjectClient) addJSONBodyReader(request *http.Request, body io.Reader) {
	request.Header.Add("Content-Type", "application/json; charset=utf-8")
	request.Body = ioutil.NopCloser(body)
}

func (p ProjectClient) addJSONBody(request *http.Request, body string) {
	p.addJSONBodyReader(request, strings.NewReader(body))
}
