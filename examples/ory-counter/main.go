package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/adnaan/fir"
	ory "github.com/ory/client-go"
)

type App struct {
	ory *ory.APIClient
	// save the cookies for any upstream calls to the Ory apis
	cookies string
	// save the session to display it on the dashboard
	session *ory.Session
}

func (app *App) sessionMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		log.Printf("handling middleware request\n")

		// set the cookies on the ory client
		var cookies string

		// this example passes all request.Cookies
		// to `ToSession` function
		//
		// However, you can pass only the value of
		// ory_session_projectid cookie to the endpoint
		cookies = request.Header.Get("Cookie")

		// check if we have a session
		session, _, err := app.ory.V0alpha2Api.ToSession(request.Context()).Cookie(cookies).Execute()
		if (err != nil && session == nil) || (err == nil && !*session.Active) {
			// this will redirect the user to the managed Ory Login UI
			http.Redirect(writer, request, "/.ory/self-service/login/browser", http.StatusSeeOther)
			return
		}
		app.cookies = cookies
		app.session = session
		// continue to the requested page (in our case the Dashboard)
		next.ServeHTTP(writer, request)
		return
	}
}

type CounterView struct {
	fir.DefaultView
	count int32
}

func (c *CounterView) Inc() int32 {
	atomic.AddInt32(&c.count, 1)
	return atomic.LoadInt32(&c.count)
}
func (c *CounterView) Dec() int32 {
	atomic.AddInt32(&c.count, -1)
	return atomic.LoadInt32(&c.count)
}

func (c *CounterView) Value() int32 {
	return atomic.LoadInt32(&c.count)
}

func (c *CounterView) Content() string {
	return `<!DOCTYPE html>
	<html lang="en">
	
	<head>
		<title>{{.app_name}}</title>
		<meta charset="UTF-8">
		<meta name="description" content="A fir counter app">
		<meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=5.0, minimum-scale=1.0">
		<link rel="preload" as="style" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/5.15.2/css/all.min.css"
			integrity="sha512-HK5fgLBL+xu6dm/Ii3z4xhlSUyZgTT9tuc/hSrtw6uzJOvgRr2a9jyxxT1ely+B+xFAmJKVSTbpM/CuL7qxO8w=="
			crossorigin="anonymous" />
		<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bulma@0.9.4/css/bulma.min.css" />
		<script defer src="http://localhost:8000/cdn.js"></script>
	</head>
	
	<body>
		<div class="my-6" style="height: 500px">
			<div class="columns is-mobile is-centered is-vcentered">
				<div x-data class="column is-one-third-desktop has-text-centered is-narrow">
					<div>
						{{block "count" .}}<div id="count">{{.count}}</div>{{end}}
						<button class="button has-background-primary" @click="$fir.emit('inc')">+
						</button>
						<button class="button has-background-primary" @click="$fir.emit('dec')">-
						</button>
					</div>
				</div>
			</div>
		</div>
		<footer class="footer">
			<div class="content has-text-centered">
				<p>
					{{.app_name}}, 2022
				</p>
			</div>
		</footer>
	</body>
	
	</html>`
}

func (c *CounterView) OnRequest(_ http.ResponseWriter, _ *http.Request) (fir.Status, fir.Data) {
	return fir.Status{Code: 200}, fir.Data{
		"count": c.Value(),
	}
}

func (c *CounterView) OnPatch(event fir.Event) (fir.Patchset, error) {
	switch event.ID {
	case "inc":
		return fir.Patchset{
			fir.Morph{
				Selector: "#count",
				Template: "count",
				Data:     fir.Data{"count": c.Inc()}}}, nil

	case "dec":
		return fir.Patchset{
			fir.Morph{
				Selector: "#count",
				Template: "count",
				Data:     fir.Data{"count": c.Dec()}}}, nil
	default:
		log.Printf("warning:handler not found for event => \n %+v\n", event)
	}

	return nil, nil
}

func main() {
	proxyPort := os.Getenv("PROXY_PORT")
	if proxyPort == "" {
		proxyPort = "4000"
	}
	c := ory.NewConfiguration()
	c.Servers = ory.ServerConfigurations{{URL: fmt.Sprintf("http://localhost:%s/.ory", proxyPort)}}

	app := &App{
		ory: ory.NewAPIClient(c),
	}

	controller := fir.NewController("fir-ory-counter", fir.DevelopmentMode(true))
	http.Handle("/", app.sessionMiddleware(controller.Handler(&CounterView{})))
	log.Println("listening on http://localhost:9867")
	http.ListenAndServe(":9867", nil)
}
