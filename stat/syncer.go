package stat

import (
	"sandwich/config"
	"sandwich/log"
	"time"
)

func InitStatSyncer() {
	initCacheFromFile()
	cfg := config.Get()
	du := cfg.Stat.SyncDuration
	if du == 0 {
		du = 60
	}

	sdu := cfg.Stat.SaveDuration
	if sdu == 0 {
		sdu = 60
	}

	go func() {
		ticker := time.NewTicker(time.Second * time.Duration(du))
		defer ticker.Stop()
		for range ticker.C {
			log.Info("running stat syncer")
			go syncStat()
		}
	}()

	go func() {
		ticker := time.NewTicker(time.Minute * time.Duration(sdu))
		defer ticker.Stop()
		for range ticker.C {
			log.Info("save stat to file")
			go SaveStat(cfg.Stat.SaveFile)
		}
	}()
}
