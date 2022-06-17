package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/adnaan/fir"
	"github.com/timshannon/bolthold"
)

type Todo struct {
	ID        uint64 `json:"id" boltholdKey:"ID"`
	Text      string `json:"text"`
	Done      bool   `json:"done"`
	CreatedAt time.Time
}

func NewTodosView(db *bolthold.Store) *TodosView {
	return &TodosView{
		db: db,
	}
}

type TodosView struct {
	fir.DefaultView
	db *bolthold.Store
}

func (t *TodosView) Content() string {
	return "app.html"
}

func (t *TodosView) Partials() []string {
	return []string{"todos.html"}
}

func (t *TodosView) OnRequest(_ http.ResponseWriter, _ *http.Request) (fir.Status, fir.Data) {
	todos := make([]Todo, 0)
	if err := t.db.Find(&todos, &bolthold.Query{}); err != nil {
		return fir.Status{
			Code: 200,
		}, nil
	}
	b, _ := json.Marshal(todos)
	return fir.Status{Code: 200}, fir.Data{"todos": string(b)}
}

func (t *TodosView) OnPatch(event fir.Event) (fir.Patchset, error) {
	var todo Todo
	if err := event.DecodeParams(&todo); err != nil {
		return nil, err
	}

	switch event.ID {

	case "todos/new":
		if len(todo.Text) < 4 {
			return fir.Patchset{
				fir.Store{
					Name: "formData",
					Data: map[string]any{
						"textError": "Min length is 4",
					},
				},
			}, nil
		}
		if err := t.db.Insert(bolthold.NextSequence(), &todo); err != nil {
			return nil, err
		}
	case "todos/del":
		if err := t.db.Delete(todo.ID, &todo); err != nil {
			return nil, err
		}
	default:
		log.Printf("warning:handler not found for event => \n %+v\n", event)
	}
	// list updated todos
	todos := make([]Todo, 0) // important: initialise the array to return [] instead of null as a json response
	if err := t.db.Find(&todos, &bolthold.Query{}); err != nil {
		return nil, err
	}
	return fir.Patchset{
		fir.Store{
			Name: "formData",
			Data: map[string]any{
				"textError": "",
			},
		},
		fir.Store{
			Name: "todos",
			Data: todos,
		},
	}, nil
}

func main() {
	db, err := bolthold.Open("todos.db", 0666, nil)
	if err != nil {
		panic(err)
	}
	c := fir.NewController("fir-todos-x-for", fir.DevelopmentMode(true))
	http.Handle("/", c.Handler(NewTodosView(db)))
	log.Println("listening on http://localhost:9867")
	http.ListenAndServe(":9867", nil)
}
