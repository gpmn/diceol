package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/gpmn/diceol/resolver"
)

func handleException() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGABRT)
	signal.Notify(c, syscall.SIGSTOP)
	signal.Notify(c, syscall.SIGINT)
	signal.Notify(c, syscall.SIGKILL)
	signal.Notify(c, syscall.SIGSEGV)
	signal.Notify(c, syscall.SIGQUIT)
	signal.Notify(c, syscall.SIGTERM)
	signal.Notify(c, syscall.Signal(30)) // custom signal
	go func(c chan os.Signal) {
		for {
			sig := <-c
			log.Println("get signal : ", sig)
			buf := make([]byte, 1024*200)
			cnt := runtime.Stack(buf, true)
			buf = buf[:cnt]

			log.Printf(`=== BEGIN goroutine stack dump ===
	%s
	=== END goroutine stack dump ===
	`, string(buf))
			switch sig {
			case syscall.SIGTERM:
				fallthrough
			case syscall.SIGSTOP:
				fallthrough
			case syscall.SIGINT:
				fallthrough
			case syscall.SIGABRT:
				fallthrough
			case syscall.SIGKILL:
				fallthrough
			case syscall.SIGQUIT:
				fallthrough
			case syscall.SIGSEGV:
				os.Exit(1)
			}
		}
	}(c)
}

const (
	defaultContractServant = "diceonlineos"
	defaultRPCServer       = "https://w1.eosforce.cn"
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	handleException()
	url := flag.String("url", defaultRPCServer, "url of http end point")
	dbpath := flag.String("db", "", "sqlite3 db file path ")
	svt := flag.String("contract", defaultContractServant, "contract servant account")
	wkey := flag.String("wallet", "", "key to unlock wallet")
	fetchIdle := flag.Uint("idlems", 500, "fetch idle milli seconds once reach to the end of service")
	bpi := flag.Uint("bpi", 500, "milli seconds between each block produced, eosforce 3000, eos 500.")
	from := flag.Int64("from", 0, "query from which blocknum.")

	flag.Parse()

	var ctrl resolver.ResolverCtrl

	ctrl.RpcURL = *url
	ctrl.DbPath = *dbpath
	ctrl.ContractServant = *svt
	ctrl.WalletKey = *wkey
	ctrl.FetchIdleDur = time.Duration(*fetchIdle) * time.Millisecond
	ctrl.BPInterval = time.Duration(*bpi) * time.Millisecond
	ctrl.FromBlkNum = *from

	if ctrl.DbPath == "" {
		log.Printf("db param invalid")
		flag.Usage()
		os.Exit(1)
	}

	if ctrl.WalletKey == "" {
		log.Printf("wallet param invalid")
		flag.Usage()
		os.Exit(2)
	}

	//go ctrl.FetchActionsRoutine() // 只是发奖的话，resolver 不需要去查询action表
	ctrl.RunResolve()
}
