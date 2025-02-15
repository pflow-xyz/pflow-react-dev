package server

import (
	"encoding/json"
	"github.com/pflow-dev/pflow-xyz/protocol/image"
	"github.com/pflow-dev/pflow-xyz/protocol/metamodel"
	"github.com/pflow-dev/pflow-xyz/protocol/model"
	"github.com/pflow-dev/pflow-xyz/protocol/zblob"
	"html/template"
	"net/http"
)

type HandlerWithVars = func(vars map[string]string, w http.ResponseWriter, r *http.Request)

type VarsFactory = func(r *http.Request) map[string]string

func WithVars(handler HandlerWithVars, getVarsFunc VarsFactory) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handler(getVarsFunc(r), w, r)
	}
}

type BlobAccessor interface {
	Get(id int64) *zblob.Zblob
	GetByCid(cid string) *zblob.Zblob
	GetMaxId() int64
	Create(ipfsCid, base64Zipped, title, description, keywords, referrer string) (int64, error)
}

type Storage struct {
	Model   BlobAccessor
	Snippet BlobAccessor
}

type Service interface {
	IndexPage() *template.Template
	Event(eventType string, params map[string]interface{})
	GetState(r *http.Request) (state metamodel.Vector, ok bool)
	CheckForModel(hostname string, url string, referrer string) (string, bool)
}

type App struct {
	Service
	Storage
}

func (app *App) AppPage(vars map[string]string, w http.ResponseWriter, r *http.Request) {
	cid, _ := app.CheckForModel(r.Host, r.URL.String(), r.Header.Get("Referer"))
	m := model.Model{
		Zblob: &zblob.Zblob{
			IpfsCid: cid,
		},
	}
	if vars["pflowCid"] != "" {
		m = *model.FromZblob(app.Storage.Model.GetByCid(vars["pflowCid"]))
		if m.ID != 0 && m.IpfsCid == vars["pflowCid"] {
			m.MetaModel()
		}
	}
	err := app.IndexPage().ExecuteTemplate(w, "index.html", m.Zblob)
	if err != nil {
		return
	}
}

func (app *App) SvgHandler(vars map[string]string, w http.ResponseWriter, r *http.Request) {
	cid, found := app.CheckForModel(r.Host, r.URL.String(), r.Header.Get("Referer"))
	if found {
		http.Redirect(w, r, "/img/"+cid+".svg", http.StatusFound)
		return
	}
	if vars["pflowCid"] == "" {
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml ; charset=utf-8")
	m := model.FromZblob(app.Storage.Model.GetByCid(vars["pflowCid"]))
	_, mm := m.MetaModel()
	if m.IpfsCid != vars["pflowCid"] {
		return
	}
	app.Event("viewSvg", map[string]interface{}{
		"id":      m.ID,
		"ipfsCid": m.IpfsCid,
	})
	x1, y1, width, height := mm.GetViewPort()
	i := image.NewSvg(w, width, height, x1, y1, width, height)

	state, stateOk := app.GetState(r)
	if !stateOk || len(state) != len(mm.Net().Places) {
		state = mm.Net().InitialVector()
	}
	i.Render(mm, state)
}

func (app *App) JsonHandler(vars map[string]string, w http.ResponseWriter, r *http.Request) {
	mm := metamodel.New()
	cid, found := app.CheckForModel(r.Host, r.URL.String(), r.Header.Get("Referer"))
	if found {
		http.Redirect(w, r, "/src/"+cid+".json", http.StatusFound)
	} else if vars["pflowCid"] != "" {
		m := app.Storage.Model.GetByCid(vars["pflowCid"])
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		mm.UnpackFromUrl("?z=" + m.Base64Zipped)
		data, _ := json.MarshalIndent(mm.ToDeclarationObject(), "", "  ")
		_, err := w.Write(data)
		if err != nil {
			panic(err)
		}
	}
}

type ShareResponse struct {
	Cid string `json:"cid"`
	Ok  bool   `json:"ok"`
}

func (app *App) ShareHandler(vars map[string]string, w http.ResponseWriter, r *http.Request) {
	_ = vars
	// Parse JSON from the request body
	cid, found := app.CheckForModel(r.Host, r.URL.String(), r.Header.Get("Referer"))
	res := ShareResponse{
		Cid: cid,
		Ok:  found,
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	out, _ := json.Marshal(res)
	_, _ = w.Write(out)
}
