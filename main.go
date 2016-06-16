package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"strings"

	"github.com/antonholmquist/jason"
	"github.com/go-yaml/yaml"
	"github.com/gorilla/mux"
)

// Config is shared accross all the application
var Config instanceConfig

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

type instanceConfig struct {
	Host    string
	Port    int
	Metrics bool
	Debug   bool
	DumpDir string
	Hooks   map[string]string
	MMicon  string
	MMuser  string
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<h1>MatterBridge Handler</h1><div>Go bridge</div>")
}

func getHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, "<h1>MatterBridge Handler</h1><div>Go bridge : get handler</div>")
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

//Inform send message to the right channel in MM
func (c *MMController) Inform(update issueUpdate) error {

	purl, err := c.GetTarget(update.Project)
	if err != nil {
		return err
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
		return (err)
	}
	defer resp.Body.Close()

	return nil
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("received a request")
	dump, err := httputil.DumpRequest(r, true)

	if err != nil {
		http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
		return
	}
	if Config.Debug {
		var tmpfile *os.File
		tmpfile, err = ioutil.TempFile(Config.DumpDir, "example")
		if err != nil {
			log.Fatal(err)
		}
		tmpfile.Write(dump)
		tmpfile.Close()
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

	changes := make(map[string]string)
	for _, item := range items {
		field, _ := item.GetString("field")
		value, _ := item.GetString("toString")
		changes[field] = value
	}
	issue := issueUpdate{Event: event, User: user, Userurl: userurl, Summary: summary, ID: id, URL: url, Project: pname, Changes: changes}
	// We only know our top-level keys are strings

	log.Printf("%+v", issue)
	go mmpost.Inform(issue)

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
	log.Printf("config : %+v", Config)
	r.HandleFunc("/", homeHandler)
	r.HandleFunc("/hooks/", getHandler).Methods("GET")
	r.HandleFunc("/hooks/", postHandler).Methods("POST")
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe("0.0.0.0:8585", r))
}

func (c *instanceConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var aux struct {
		Hostname string            `yaml:"host"`
		Port     string            `yaml:"port"`
		Metrics  string            `yaml:"metrics"`
		Debug    string            `yaml:"debug"`
		Hooks    map[string]string `yaml:"hooks"`
		DumpDir  string            `yaml:"dumpdir"`
		MMuser   string            `yaml:"mmuser"`
		MMIcon   string            `yaml:"mmicon"`
	}
	log.Println("validating config")
	if err := unmarshal(&aux); err != nil {
		return err
	}
	if aux.Hostname == "" {
		return errors.New("Brigge config: invalid `hostname`")
	}

	port, err := strconv.Atoi(aux.Port)
	if err != nil {
		return errors.New("Bridge config: invalid `port`")
	}

	// Test Kitchen stores the port as an string
	metrics, err := strconv.ParseBool(aux.Metrics)
	if err != nil {
		return errors.New("Bridge config: invalid `metrics`")
	}
	debug, err := strconv.ParseBool(aux.Debug)
	if err != nil {
		return errors.New("Bridge config: invalid `debug`")
	}
	upperHooks := make(map[string]string)
	for key, value := range aux.Hooks {
		upperHooks[strings.ToUpper(key)] = value
	}
	c.Host = aux.Hostname
	c.Port = port
	c.Metrics = metrics
	c.Hooks = upperHooks
	c.Debug = debug

	c.DumpDir = aux.DumpDir
	c.MMicon = aux.MMIcon
	c.MMuser = aux.MMuser

	log.Println("config validated")
	return nil
}
