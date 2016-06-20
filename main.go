package main

import (
	"expvar"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/tixu/mmjiraserver/jira"
	"github.com/tixu/mmjiraserver/mmcontroller"
	"github.com/tixu/mmjiraserver/utils"

	"net/http"
	"strconv"
	"time"

	metrics "github.com/armon/go-metrics"
	"github.com/go-yaml/yaml"
	"github.com/gorilla/mux"
)

// Config is shared accross all the application
var (
	Config        InstanceConfig
	hitsperminute = expvar.NewInt("hits_per_minute")
	inm           = metrics.NewInmemSink(10*time.Millisecond, 50*time.Millisecond)
)

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

//---------------------------------------------------------------------------------

var mmpost *mmcontroller.MMController

// GetTarget retrieve the hook assigned to a projet, return an error in anyother case

func postHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("received a request")

	inm.IncrCounter([]string{"request", "jira"}, 1)
	if Config.Debug {
		if err := utils.DumpRequest(r, Config.DumpDir); err != nil {
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
	go mmpost.Analyse(mmpost.Inform(issue))

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
	mmpost, err = mmcontroller.NewController(Config.MMicon, Config.MMuser, Config.Hooks)
	if err != nil {
		panic(err)
	}

	log.Printf("config : %+v", Config)
	r.HandleFunc("/", homeHandler)
	r.HandleFunc("/hooks/", getHandler).Methods("GET")
	r.HandleFunc("/hooks/", postHandler).Methods("POST")
	http.Handle("/", r)
	endpoint := Config.Host + ":" + strconv.Itoa(Config.Port)
	log.Fatal(http.ListenAndServe(endpoint, r))
}
