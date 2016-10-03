package blackout

import (
        "net/http"
)

var (
	//validEmail = regexp.MustCompile("^.*@.*\\.(com|org|in|mail|io)$")
)


func init() {

        r := NewRouter()
        http.Handle("/", r)

}



