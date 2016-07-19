package mmcontroller

import (
	"bytes"
	"encoding/json"
	metrics "github.com/rcrowley/go-metrics"
	"github.com/tixu/mmjira/jira"
	"github.com/uber-go/zap"
	"io"
	"net/http"
	"strings"
)


// Request  is the message that will be send to mm
type Request struct {
	Icon string `json:"icon_url"`
	User string `json:"username"`
	Text string `json:"text"`
}

//Response is the response from MM
type Response struct {
	Project    string `json:"project"`
	EndPoint   string `json:"endpoint"`
	ID         string `json:"jiraid"`
	Status     string `json:"status"`
	StatusCode int    `json:"statuscode"`
	Error      string `json:"error"`
}

//Controller is repsonsible to handle the communication towards MM
type Controller struct {
	l         zap.Logger
	icon      string
	name      string
	converter jira.Controller
	hooks     map[string]string
	reg       metrics.Registry
}

// NewController is used to create a MMController
func NewController(icon string, name string, hooks map[string]string, debug bool, reg metrics.Registry) (m *Controller, err error) {
	m = new(Controller)
	m.icon = icon
	m.name = name
	m.hooks = hooks
	m.l = zap.NewJSON(zap.ErrorLevel)
	if debug {
		m.l.SetLevel(zap.DebugLevel)
	}
	m.converter = jira.FNew()
	m.reg = reg
	return m, nil
}

//Inform send message to the right channel in MM
func (c *Controller) Inform(update jira.IssueEvent) <-chan Response {

	c.l.Info("about to inform")
	count := metrics.GetOrRegisterCounter("inform.request.total", c.reg)
	count.Inc(1)
	ch := make(chan Response)
	go func() {
		response := Response{Project: strings.ToLower(update.Project), ID: update.ID}
		count := metrics.GetOrRegisterCounter("inform.request."+response.Project, c.reg)
		count.Inc(1)

		purl := c.hooks[strings.ToLower(update.Project)]
		if purl == "" {
			response.Status = "1002 - not mapped"
			response.StatusCode = 1002
			ch <- response
			return
		}
		response.EndPoint = purl
		c.l.Debug("about to post", zap.String("post url", purl))
		buff, err := c.converter.Convert(update)
		if err != nil {
			response.Error = err.Error()
			response.Status = "1003 - not templated"
			response.StatusCode = 1003
			ch <- response
			return
		}

		s2, _ := json.Marshal(&Request{User: c.name, Icon: c.icon, Text: string(buff.Bytes())})
		req, err := http.NewRequest("POST", purl, bytes.NewBuffer(s2))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}

		resp, err := client.Do(req)

		if err != nil {
			response.Error = err.Error()
			ch <- response
			return
		}
		response.Error = ""
		response.Status = resp.Status
		response.StatusCode = resp.StatusCode

		ch <- response
		close(ch)
	}()
	return ch
}

// Convert  converts a
func (c *Controller) Convert(update jira.IssueEvent) (text *bytes.Buffer, err error) {
	return c.converter.Convert(update)
}

// Create an issueEvent
func (c *Controller) Create(reader io.Reader) (i jira.IssueEvent, err error) {
	return c.converter.Create(reader)
}

//Analyse the response from mm
func (c *Controller) Analyse(in <-chan Response) {

	count := metrics.GetOrRegisterCounter("analyse.response.total", c.reg)
	count.Inc(1)

	response := <-in
	if response.StatusCode != 200 {
		n := "analyse.response." + response.Project + ".error"
		count := metrics.GetOrRegisterCounter(n, c.reg)
		count.Inc(1)
	} else {
		n := "analyse.response." + response.Project + ".ok"
		count := metrics.GetOrRegisterCounter(n, c.reg)
		count.Inc(1)
	}
	c.l.Info("response received", zap.Object("response", response))
}
