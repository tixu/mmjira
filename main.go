package main

import (
	"fmt"
	"strings"
	"strconv"

	"io/ioutil"
	"html/template"
	"net/http"

  	metrics "github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/exp"

	"github.com/go-yaml/yaml"
	"github.com/gorilla/mux"
	"github.com/fatih/structs"
	"github.com/uber-go/zap"

	
	"github.com/tixu/mmjira/mmcontroller"
	"github.com/tixu/mmjira/utils"

)

// Page is structuring information
type Page struct {
	T string
	C map[string]int64
}

// MMJira is the heart of the bots
type MMJira struct {
	c *InstanceConfig
	m mmcontroller.MattermostBridge
	r *mux.Router
	l zap.Logger
	reg metrics.Registry
}

func (b MMJira) homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<h1>MatterBridge Handler</h1><div>Go bridge</div>")
}

func (b MMJira) getHandler(w http.ResponseWriter, r *http.Request) {
	b.l.Debug("got a request to the hetHandler")
  //increasing the counter
	c:=metrics.GetOrRegisterCounter("hooks.get", b.reg)
	c.Inc(1)
  // computing the map
  cv  :=make(map[string]int64)
	anom := func (k string, v interface{}) {
		if tmp, ok :=v.(metrics.Counter); ok {
			 cv[k]=tmp.Count()
			}
		}
	b.reg.Each(anom)
	//t:=template.New("info")
	var err error
	t, err := template.ParseFiles("templates/info.html")
	if err !=nil {
   		http.Error(w, err.Error(),http.StatusInternalServerError)
		return
			}
	p := Page{T:"Overview",C:cv}
	if err := t.Execute(w, p);err !=nil {
		http.Error(w, err.Error(),http.StatusInternalServerError)
	}
}

func (b MMJira) configGetHandler(w http.ResponseWriter, r *http.Request){
  c:=structs.Map(b.c)
	p :=struct {
		T string
		C map[string]interface{}
	}{T:"Configuration",C:c }

	t, err := template.ParseFiles("templates/config.html")
	if err !=nil {
		http.Error(w, err.Error(),http.StatusInternalServerError)
		return
			}

	if err := t.Execute(w, p);err !=nil {
		http.Error(w, err.Error(),http.StatusInternalServerError)
	}


}
// GetTarget retrieve the hook assigned to a projet, return an error in anyother case
func (b MMJira) postHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hookid := strings.ToLower(vars["hookid"])
	b.l.Info("project", zap.String("hook",hookid))
	if (b.c.Hooks[hookid] ==""){
		  c:=metrics.GetOrRegisterCounter("hooks.post.unknown.project", b.reg)
		  c.Inc(1)
			http.Error(w, "unknwon project", http.StatusBadRequest)
			return
		}
	b.l.Debug("received a request")
	c:=metrics.GetOrRegisterCounter("hooks.received."+hookid, b.reg)
	c.Inc(1)
	if b.c.Debug {
		if err := utils.DumpRequest(r, b.c.DumpDir); err != nil {
			b.l.Info("unable to dump the request in the directory",zap.String("Directory",b.c.DumpDir))
		}
	}
	issue, err := b.m.Create(r.Body)
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

  b = MMJira{l: zap.NewJSON(zap.DebugLevel), reg : metrics.NewRegistry()}
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
	mmpost, err := mmcontroller.NewController(b.c.MMicon,b.c.MMuser,b.c.Hooks,b.c.Debug,metrics.NewPrefixedChildRegistry(b.reg, "mmc."))
	if err != nil {
		panic(err)
	}

	b.m = mmpost
	b.l.Debug("outputting config", zap.Object("config", b.c))
	b.r = mux.NewRouter()
	b.r.HandleFunc("/", b.homeHandler)
	b.r.HandleFunc("/hooks/", b.getHandler).Methods("GET")
	b.r.HandleFunc("/hooks/{hookid}", b.postHandler).Methods("POST")
	b.r.Handle("/metrics",exp.ExpHandler(b.reg))
	b.r.HandleFunc("/config/", b.configGetHandler).Methods("GET")

	return b

}

func main() {
	b:= newBot()
	b.start()
}
