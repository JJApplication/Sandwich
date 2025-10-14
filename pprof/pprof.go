package pprof

import (
	"fmt"
	"net/http"
	"net/http/pprof"
)

func InitPProf(pc struct {
	Enable bool `yaml:"enable" json:"enable"`
	Port   int  `yaml:"port" json:"port"`
}) {
	if pc.Enable {
		go func() {
			http.HandleFunc("/debug/pprof/", pprof.Index)
			http.ListenAndServe(fmt.Sprintf(":%d", pc.Port), nil)
		}()
	}
}
