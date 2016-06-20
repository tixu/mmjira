package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"expvar"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"time"

	"github.com/antonholmquist/jason"
	metrics "github.com/armon/go-metrics"
	"github.com/go-yaml/yaml"
	"github.com/gorilla/mux"
	"github.com/pkg/profile"
)

// Config is shared accross all the application
var (
	Config        InstanceConfig
	hitsperminute = expvar.NewInt("hits_per_minute")
	inm           = metrics.NewInmemSink(10*time.Millisecond, 50*time.Millisecond)
)

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

type issueUpdate struct {
	Event   string
	User    string
	Userurl string
	ID      string
	URL     string
	Summary string
	Project string
	Changes map[string]string
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<h1>MatterBridge Handler</h1><div>Go bridge</div>")
}

func getHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, "<h1>MatterBridge Handler</h1><div>Go bridge : get handler</div>")
	data := inm.Data()

	intvM := data[0]
	intvM.RLock()
	log.Printf("%+v", inm)
	intvM.RUnlock()
}

// MMController is repsonsible to handle the communication towards MM
type MMController struct {
	icon       string
	name       string
	hooks      map[string]string
	mmtemplate *template.Template
}

//---------------------------------------------------------------------------------

var mmpost *MMController

// NewController is used to create a MMController
func NewController(icon string, name string, hooks map[string]string) (m *MMController, err error) {
	m = new(MMController)
	m.icon = icon
	m.name = name
	m.hooks = hooks

	t := template.New("Event")
	t, err = t.Parse(tmpl)
	if err != nil {
		log.Panic(err)
		return m, err
	}
	m.mmtemplate = t
	return m, nil

}

// GetTarget retrieve the hook assigned to a projet, return an error in anyother case
func (c *MMController) GetTarget(project string) (response string, err error) {
	response = c.hooks[strings.ToUpper(project)]
	if response == "" {
		err = errors.New("project is not mapped")
	}
	return response, err
}

//Analyse the response from mm
func (c *MMController) Analyse(in <-chan MMresponse) {

	response := <-in
	log.Printf("%+v", response)
	inm.IncrCounter([]string{strconv.Itoa(response.StatusCode), response.Project, response.ID}, 1)
}

//Inform send message to the right channel in MM
func (c *MMController) Inform(update issueUpdate) <-chan MMresponse {
	ch := make(chan MMresponse)
	go func() {
		response := MMresponse{Project: update.Project, ID: update.ID}
		purl, err := c.GetTarget(update.Project)
		if err != nil {
			response.Error = err.Error()
			response.Status = "1002 - not mapped"
			response.StatusCode = 1002
			ch <- response
			return
		}
		log.Printf("about to post %s", purl)
		buff := bytes.NewBufferString("")
		c.mmtemplate.Execute(buff, update)

		s2, _ := json.Marshal(&Mmrequest{User: c.name, Icon: c.icon, Text: string(buff.Bytes())})

		req, err := http.NewRequest("POST", purl, bytes.NewBuffer(s2))

		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}

		resp, err := client.Do(req)

		if err != nil {
			response.Error = err.Error()
			response.EndPoint = purl
			response.Error = err.Error()
			response.Status = resp.Status
			response.StatusCode = resp.StatusCode
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

func postHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("received a request")

	inm.IncrCounter([]string{"request", "jira"}, 1)
	if Config.Debug {
		if err := dumpRequest(r); err != nil {
			http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
		}
	}
	v, err := jason.NewObjectFromReader(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprint(err), http.StatusBadRequest)
		return
	}

	user, _ := v.GetString("user", "name")
	userurl, _ := v.GetString("user", "avatarUrls", "24x24")
	summary, _ := v.GetString("issue", "fields", "summary")
	event, _ := v.GetString("webhookEvent")
	id, _ := v.GetString("issue", "id")
	url, _ := v.GetString("issue", "self")
	pname, _ := v.GetString("issue", "fields", "project", "name")
	items, err := v.GetObjectArray("changelog", "items")
	if err != nil {
		http.Error(w, fmt.Sprint(err), http.StatusBadRequest)
		return
	}
	inm.IncrCounter([]string{pname, id}, 1)

	changes := make(map[string]string)
	for _, item := range items {
		field, _ := item.GetString("field")
		value, _ := item.GetString("toString")
		changes[field] = value
	}
	issue := issueUpdate{Event: event, User: user, Userurl: userurl, Summary: summary, ID: id, URL: url, Project: pname, Changes: changes}
	// We only know our top-level keys are strings

	log.Printf("%+v", issue)
	go mmpost.Analyse(mmpost.Inform(issue))

}

func dumpRequest(r *http.Request) (err error) {

	dump, err := httputil.DumpRequest(r, true)

	if err != nil {
		return err
	}
	if Config.Debug {

		tmpfile, err := ioutil.TempFile(Config.DumpDir, "example")
		if err != nil {
			return err
		}
		tmpfile.Write(dump)
		tmpfile.Close()
	}
	return nil
}

func main() {

	r := mux.NewRouter()
	data, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Panic(err)
	}

	if err = yaml.Unmarshal(data, &Config); err != nil {
		log.Panic(err)
	}
	mmpost, err = NewController(Config.MMicon, Config.MMuser, Config.Hooks)
	if err != nil {
		panic(err)
	}
	// activating
	switch Config.Profile {
	case "cpu":
		defer profile.Start(profile.ProfilePath("."), profile.CPUProfile).Stop()
	case "mem":
		defer profile.Start(profile.ProfilePath("."), profile.MemProfile).Stop()
	case "block":
		defer profile.Start(profile.ProfilePath("."), profile.BlockProfile).Stop()
	default:
		// do nothing
	}

	log.Printf("config : %+v", Config)
	r.HandleFunc("/", homeHandler)
	r.HandleFunc("/hooks/", getHandler).Methods("GET")
	r.HandleFunc("/hooks/", postHandler).Methods("POST")
	http.Handle("/", r)
	endpoint := Config.Host + ":" + strconv.Itoa(Config.Port)
	log.Fatal(http.ListenAndServe(endpoint, r))
}
