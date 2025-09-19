package prehandler

import "net/http"

type PreHandler interface {
	Access(r *http.Request) error
}
