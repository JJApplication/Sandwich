package stat

import (
	"context"
	"fmt"
	"net/http"
	"sandwich/config"
	"sandwich/json"
	"sandwich/log"
)

// stat server

type StatServer struct {
	Enabled bool
	Addr    string
	Port    int

	logger *log.Log
	server *http.Server
}

func NewStatServer(c config.StatConfig, l *log.Log) *StatServer {
	return &StatServer{
		Enabled: c.Enabled,
		Addr:    c.Host,
		Port:    c.Port,

		logger: l,
		server: newServer(c.Host, c.Port),
	}
}

func (s *StatServer) Start() error {
	if !s.Enabled {
		return nil
	}
	log.InfoF("开启状态统计服务: %s:%d", s.Addr, s.Port)

	go func() {
		err := s.server.ListenAndServe()
		if err != nil {
			log.ErrorF("Stat server listen err: %v\n", err)
		}
	}()

	return nil
}

func (s *StatServer) Stop() error {
	if !s.Enabled {
		return nil
	}
	return s.server.Shutdown(context.Background())
}

func newServer(host string, port int) *http.Server {
	svr := &http.Server{
		Addr: fmt.Sprintf("%s:%d", host, port),
	}
	mux := http.NewServeMux()

	registerMux(mux)
	svr.Handler = mux
	return svr
}

func registerMux(mux *http.ServeMux) {
	mux.HandleFunc("/api/stat", func(w http.ResponseWriter, r *http.Request) {
		result := make(map[string]int64)
		result["total"] = Get(Total)
		result["api"] = Get(API)
		result["static"] = Get(Static)
		result["fail"] = Get(Fail)
		result["today"] = Get(Today)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		data, err := json.Marshal(result)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(data)
	})

	mux.HandleFunc("/api/geo", func(w http.ResponseWriter, r *http.Request) {
		result := GetGeoData()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Write(result)
	})

	mux.HandleFunc("/api/domain", func(w http.ResponseWriter, r *http.Request) {
		result := GetDomainStat()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Write(result)
	})
}
