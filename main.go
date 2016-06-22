package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/tixu/mmjira/jira"
	"github.com/tixu/mmjira/mmcontroller"
	"github.com/tixu/mmjira/utils"

	"net/http"
	"strconv"

	"github.com/go-yaml/yaml"
	"github.com/gorilla/mux"
)

// Config is shared accross all the application

func (b MMJira) homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<h1>MatterBridge Handler</h1><div>Go bridge</div>")
}

func (b MMJira) getHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, "<h1>MatterBridge Handler</h1><div>Go bridge : get handler</div>")
}

// MMJira is the heart of the bots
type MMJira struct {
	c *InstanceConfig
	m *mmcontroller.MMController
}

// GetTarget retrieve the hook assigned to a projet, return an error in anyother case

func (b MMJira) postHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("received a request")

	if b.c.Debug {
		if err := utils.DumpRequest(r, b.c.DumpDir); err != nil {
			http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
		}
	}
	issue, err := jira.New(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprint(err), http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprint(err), http.StatusBadRequest)
		return
	}

	// We only know our top-level keys are strings

	log.Printf("%+v", issue)

	ch := b.m.Inform(issue)
	go b.m.Analyse(ch)
}

func main() {

	r := mux.NewRouter()
	data, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Panic(err)
	}
	var config InstanceConfig
	if err = yaml.Unmarshal(data, &config); err != nil {
		log.Panic(err)
	}
	b := MMJira{c: &config}

	mmpost, err := mmcontroller.NewController(b.c.MMicon, b.c.MMuser, b.c.Hooks)
	if err != nil {
		panic(err)
	}

	b.m = mmpost
	log.Printf("config : %+v", b.c)
	r.HandleFunc("/", b.homeHandler)

	r.HandleFunc("/hooks/", b.getHandler).Methods("GET")
	r.HandleFunc("/hooks/", b.postHandler).Methods("POST")
	http.Handle("/", r)
	endpoint := b.c.Host + ":" + strconv.Itoa(b.c.Port)
	log.Fatal(http.ListenAndServe(endpoint, r))
}
