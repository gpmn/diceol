package controllers

import (
	"log"
	"os"
	"time"

	"github.com/gpmn/diceol/resolver"
	"github.com/revel/revel"
)

var rsvCtrl resolver.ResolverCtrl

// InitResolver :
func InitResolver() {
	rsvCtrl.RpcURL = revel.Config.StringDefault("app.RpcURL", "")
	rsvCtrl.DbPath = revel.Config.StringDefault("app.DbPath", "")
	rsvCtrl.ContractServant = revel.Config.StringDefault("app.ContractServant", "")
	rsvCtrl.FetchIdleDur = time.Duration(revel.Config.IntDefault("app.FetchIdleDur", 30)) * time.Millisecond
	rsvCtrl.BPInterval = time.Duration(revel.Config.IntDefault("app.FetchIdleDur", 29)) * time.Millisecond

	if rsvCtrl.RpcURL == "" {
		log.Printf("InitResolver - missing app.RpcURL, exit process")
		os.Exit(1)
		return
	}

	if rsvCtrl.DbPath == "" {
		log.Printf("InitResolver - missing app.DbPath, exit process")
		os.Exit(2)
		return
	}

	if rsvCtrl.ContractServant == "" {
		log.Printf("InitResolver - missing app.ContractServant, exit process")
		os.Exit(3)
		return
	}

	go rsvCtrl.RunFetchActions()
}
