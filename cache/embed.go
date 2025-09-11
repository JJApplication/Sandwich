package cache

import _ "embed"

var (
	//go:embed static/forbidden.html
	ForbiddenPage []byte
	//go:embed static/unavailable.html
	UnavailablePage []byte
)
