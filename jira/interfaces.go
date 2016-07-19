package jira

import (
	"bytes"
	"io"
)

// Controller describes the behavior expected from Jira issue receiver
type Controller interface {
	Convert(i IssueEvent) (text *bytes.Buffer, err error)
	Create(reader io.Reader) (i IssueEvent, err error)
}
