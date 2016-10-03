package blackout

import (
	"net/http"
        "github.com/gorilla/mux"
        "regexp"
)

var validPath = regexp.MustCompile(`^/(cal|map|paksha|logs)?/?(.*)$`)

func NewRouter() *mux.Router {

        router := mux.NewRouter().StrictSlash(true)
        for _, route := range routes {
                var handler http.Handler
                handler = makeHandler(route.HandlerFunc)
                //handler = Logger(handler, route.Name)

                router.Methods(route.Method).Path(route.Pattern).Name(route.Name).Handler(handler)
        }
        return router
}

func makeHandler(fn func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                m := validPath.FindStringSubmatch(r.URL.Path)
                if m == nil {
                        http.NotFound(w,r)
                        return
                }
                fn(w,r)
        }
}
