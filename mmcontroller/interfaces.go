package mmcontroller

import(
  	"github.com/tixu/mmjira/jira"
)

// MattermostBridge  is controlling the integration with Mattermost.

type MattermostBridge interface {
	jira.Controller
	Analyse(in <-chan Response)
	Inform(update jira.IssueEvent) <-chan Response
}
