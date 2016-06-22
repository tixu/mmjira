package main

import (
	"fmt"
	"io/ioutil"
	"github.com/uber-go/zap"

	"github.com/tixu/mmjira/jira"
	"github.com/tixu/mmjira/mmcontroller"
	"github.com/tixu/mmjira/utils"

	"net/http"
	"strconv"

	"github.com/go-yaml/yaml"
	"github.com/gorilla/mux"
)




// MMJira is the heart of the bots
type MMJira struct {
	c *InstanceConfig
	m *mmcontroller.MMController
	r *mux.Router
	l zap.Logger
}

func (b MMJira) homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<h1>MatterBridge Handler</h1><div>Go bridge</div>")
}

func (b MMJira) getHandler(w http.ResponseWriter, r *http.Request) {
	b.l.Debug("got a request to the hetHandler")
	fmt.Fprintf(w, "<h1>MatterBridge Handler</h1><div>Go bridge : get handler</div>")
}


// GetTarget retrieve the hook assigned to a projet, return an error in anyother case
func (b MMJira) postHandler(w http.ResponseWriter, r *http.Request) {
	b.l.Debug("received a request")

	if b.c.Debug {
		if err := utils.DumpRequest(r, b.c.DumpDir); err != nil {
			b.l.Info("unable to dump the request in the directory",zap.String("Directory",b.c.DumpDir))

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

	b.l.Debug("sending",zap.Object("issue", issue))

	ch := b.m.Inform(issue)
	go b.m.Analyse(ch)
}

func (b MMJira) start(){
	http.Handle("/", b.r)

	endpoint := b.c.Host + ":" + strconv.Itoa(b.c.Port)
	b.l.Fatal("error server",zap.Error(http.ListenAndServe(endpoint, b.r)))
}

func newBot() (b MMJira){

  b = MMJira{l: zap.NewJSON(zap.DebugLevel)}
	data, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		b.l.Panic("not able to read the file",zap.Error(err))
	}
	var config InstanceConfig
	if err = yaml.Unmarshal(data, &config); err != nil {
		b.l.Panic("not able to marshal the file",zap.Error(err))
	}
	b.c= &config
	if (!b.c.Debug){
		b.l.SetLevel(zap.ErrorLevel)
		}
	mmpost, err := mmcontroller.NewController(b.c.MMicon, b.c.MMuser, b.c.Hooks,b.c.Debug)
	if err != nil {
		panic(err)
	}

	b.m = mmpost
	b.l.Debug("outputting config", zap.Object("config", b.c))

	b.r = mux.NewRouter()
	b.r.HandleFunc("/", b.homeHandler)

	b.r.HandleFunc("/hooks/", b.getHandler).Methods("GET")
	b.r.HandleFunc("/hooks/", b.postHandler).Methods("POST")
  return b

}

func main() {
	b:= newBot()
	b.start()
}
