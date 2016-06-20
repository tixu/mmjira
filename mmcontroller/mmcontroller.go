package mmcontroller

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/tixu/mmjiraserver/jira"
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
	icon  string
	name  string
	hooks map[string]string
}

// NewController is used to create a MMController
func NewController(icon string, name string, hooks map[string]string) (m *MMController, err error) {
	m = new(MMController)
	m.icon = icon
	m.name = name
	m.hooks = hooks

	return m, nil

}

//Inform send message to the right channel in MM
func (c *MMController) Inform(update jira.IssueEvent) <-chan MMresponse {
	ch := make(chan MMresponse)
	go func() {
		response := MMresponse{Project: update.Project, ID: update.ID}
		purl := c.hooks[strings.ToUpper(update.Project)]
		if purl == "" {
			response.Status = "1002 - not mapped"
			response.StatusCode = 1002
			ch <- response
			return
		}
		log.Printf("about to post %s", purl)
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
			response.EndPoint = purl
			response.Error = err.Error()

			ch <- response
			return
		}
		response.Error = ""
		response.EndPoint = purl
		response.Status = resp.Status
		response.StatusCode = resp.StatusCode

		ch <- response
	}()
	return ch
}

//Analyse the response from mm
func (c *MMController) Analyse(in <-chan MMresponse) {

	response := <-in
	log.Printf("%+v", response)

}
