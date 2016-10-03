package blackout

import (
	"net/http"
)

type Route struct {

        Name string
        Method string
        Pattern string
        HandlerFunc http.HandlerFunc

}

type Routes []Route

var routes = Routes {
                Route{
                        "Index",
                        "GET",
                        "/",
                        Index,
                },
                Route{
                        "Cal",
                        "GET",
                        "/cal",
                        Cal,
                },
                Route{
                        "Moon",
                        "GET",
                        "/moon",
                        Moon,
                },
                Route{
                        "Map",
                        "GET",
                        "/map",
                        Map,
                },
                Route{
                        "Paksha",
                        "GET",
                        "/paksha",
                        Paksha,
                },
                Route{
                        "Logs",
                        "GET",
                        "/logs",
                        Logs,
                },
}
