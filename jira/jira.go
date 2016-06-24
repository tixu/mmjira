package jira

import (
	"bytes"
	"html/template"
	"io"
	"log"
  "strings"
	"github.com/antonholmquist/jason"
)

// Test to validate

const tmpl = `@channel
# JIRA {{.ID}}
![profileimae]({{.Userurl}}) **{{.User}}** has performed **{{.Event}}** on Jira [{{.ID}} ]({{.URL}}) from the project **{{.Project}}**.
Summary of {{.ID}} is :

"{{.Summary}}"

## Changes
{{ range $key, $value := .Changes }}
### Changes on  {{ $key }}
{{ $value }}
{{ end }}
-------------------------------------------------------------------------------
`

var t *template.Template

func init() {
	t = template.New("Event")
	var err error
	t, err = t.Parse(tmpl)
	if err != nil {
		log.Fatal(err)

	}
}

// IssueEvent represents the event we revceived from the JIra Incoming webhook
type IssueEvent struct {
	Event   string
	User    string
	Userurl string
	ID      string
	URL     string
	Summary string
	Project string
	Changes map[string]string
}

// Render format an IssueEvent in a simple form for mm
func (i IssueEvent) Render() (text *bytes.Buffer, err error) {
	text = bytes.NewBufferString("")
	err = t.Execute(text, i)
	return text, err
}

//New creates issue from a reader
func New(reader io.Reader) (i IssueEvent, err error) {
	v, err := jason.NewObjectFromReader(reader)
	if err != nil {

		return i, nil
	}

	user, _ := v.GetString("user", "name")
	userurl, _ := v.GetString("user", "avatarUrls", "24x24")
	summary, _ := v.GetString("issue", "fields", "summary")
	event, _ := v.GetString("webhookEvent")
	id, _ := v.GetString("issue", "id")
	url, _ := v.GetString("issue", "self")
	pname, _ := v.GetString("issue", "fields", "project", "name")
	items, err := v.GetObjectArray("changelog", "items")
	changes := make(map[string]string)
	for _, item := range items {
		field, _ := item.GetString("field")
		value, _ := item.GetString("toString")
		changes[field] = value
	}

	i = IssueEvent{Event: event, User: user, Userurl: userurl, ID: id, URL: url, Summary: summary, Project: strings.ToLower(pname), Changes: changes}
	return i, nil
}
