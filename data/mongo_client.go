/*
Create: 2022/9/7
Project: Sandwich
Github: https://github.com/landers1037
Copyright Renj
*/

// Package data
package data

import (
	"sandwich/config"
	"sandwich/log"
	"time"

	"github.com/JJApplication/octopus_meta"
	"github.com/kamva/mgm/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// mongo客户端

type App struct {
	Meta octopus_meta.App
}

type DaoAPP struct {
	mgm.DefaultModel `bson:",inline"`
	App              `bson:"app"`
}

func (a *DaoAPP) CollectionName() string {
	return "microservice"
}

func InitMongo() {
	cf := config.Get()
	log.Info("init mongodb")
	err := mgm.SetDefaultConfig(&mgm.Config{CtxTimeout: 1 * time.Second}, cf.Database.Mongo.Database, options.Client().ApplyURI(cf.Database.Mongo.URL))
	if err != nil {
		log.ErrorF("failed to connect to mongo: %s\n", err.Error())
		return
	}
}

// GetAppFromMongo 获取域名映射表
func getAppFromMongo() []DaoAPP {
	var data []DaoAPP
	err := mgm.Coll(&DaoAPP{}).SimpleFind(&data, bson.M{})
	if err != nil {
		log.ErrorF("get data from mongo failed: %s\n", err.Error())
		return nil
	}
	return data
}
