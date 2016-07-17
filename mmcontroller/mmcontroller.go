package mmcontroller

import (
	"bytes"
	"encoding/json"
	"github.com/tixu/mmjira/jira"
	"github.com/uber-go/zap"
	metrics "github.com/rcrowley/go-metrics"
	"net/http"
	"strings"
)

// Mmrequest is the message that will be send to mm
type Mmrequest struct {
	Icon string `json:"icon_url"`
	User string `json:"username"`
	Text string `json:"text"`
}

//MMresponse is the response from MM
type MMresponse struct {
	Project    string `json:"project"`
	EndPoint   string `json:"endpoint"`
	ID         string `json:"jiraid"`
	Status     string `json:"status"`
	StatusCode int    `json:"statuscode"`
	Error      string `json:"error"`
}

// MMController is repsonsible to handle the communication towards MM
type MMController struct {
	l     zap.Logger
	icon  string
	name  string
	hooks map[string]string
	reg metrics.Registry
}

// NewController is used to create a MMController
func NewController(icon string, name string, hooks map[string]string, debug bool, reg metrics.Registry) (m *MMController, err error) {
	m = new(MMController)
	m.icon = icon
	m.name = name
	m.hooks = hooks
	m.l = zap.NewJSON(zap.ErrorLevel)
	if debug {
		m.l.SetLevel(zap.DebugLevel)
	}
	m.reg = reg
	return m, nil

}

//Inform send message to the right channel in MM
func (c *MMController) Inform(update jira.IssueEvent) <-chan MMresponse {

	c.l.Info("about to inform")
	count:=metrics.GetOrRegisterCounter("inform.request.total",c.reg)
	count.Inc(1)

	ch := make(chan MMresponse)
	go func() {
		response := MMresponse{Project: strings.ToLower(update.Project), ID: update.ID}
		count:=metrics.GetOrRegisterCounter("inform.request."+response.Project,c.reg)
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
		buff, err := update.Render()
		if err != nil {
			response.Error = err.Error()
			response.Status = "1003 - not templated"
			response.StatusCode = 1003
			ch <- response
			return
		}

		s2, _ := json.Marshal(&Mmrequest{User: c.name, Icon: c.icon, Text: string(buff.Bytes())})

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

//Analyse the response from mm
func (c *MMController) Analyse(in <-chan MMresponse) {

	count:=metrics.GetOrRegisterCounter("analyse.response.total",c.reg)
	count.Inc(1)

	response := <-in
	if (response.StatusCode != 200){
			n := "analyse.response."+response.Project+".error"
			count:=metrics.GetOrRegisterCounter(n,c.reg)
			count.Inc(1)
	} else {
		n := "analyse.response."+response.Project+".ok"
		count:=metrics.GetOrRegisterCounter(n,c.reg)
		count.Inc(1)
	}
	c.l.Info("response received", zap.Object("response", response))

}
