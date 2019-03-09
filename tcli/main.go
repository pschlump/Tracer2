package main

/*
Purpose:  Cli to test / use tracer2 - trader-bot.go
	-- This is really a command line client for use instead of Angular2 testing tool

--path '/listen-for' --TrxId Id --ClietTrxId Id --wait

*/

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pschlump/Go-FTL/server/lib"
	"github.com/pschlump/MiscLib"
	"github.com/pschlump/Tracer2/qdemolib"
	"github.com/pschlump/godebug"
	"github.com/pschlump/radix.v2/pubsub"
	"github.com/pschlump/radix.v2/redis" // Modified pool to have NewAuth for authorized connections
)

var Debug = flag.Bool("debug", false, "Debug flag")                      // 0
var Cfg = flag.String("cfg", "../global_cfg.json", "Configuration file") // 1

var To = flag.String("to", "", "destination send")                                         // 2
var Path = flag.String("path", "", "Path to send")                                         // 2
var TrxId = flag.String("TrxId", "", "TrxId of this program to simulate")                  // 3
var ClientTrxId = flag.String("ClientTrxId", "", "Client to Watch")                        // 4
var ListenTo = flag.String("ListenTo", "r:listen", "Pubsub to subscribe to and listen to") // 4

var Wait = flag.Bool("wait", false, "Wait for data") // 5

func RedisClient() (client *redis.Client, conFlag bool) {
	var err error
	client, err = redis.Dial("tcp", qdemolib.ServerGlobal.RedisConnectHost+":"+qdemolib.ServerGlobal.RedisConnectPort)
	if err != nil {
		log.Fatal(err)
	}
	if qdemolib.ServerGlobal.RedisConnectAuth != "" {
		err = client.Cmd("AUTH", qdemolib.ServerGlobal.RedisConnectAuth).Err
		if err != nil {
			log.Fatal(err)
		} else {
			conFlag = true
		}
	} else {
		conFlag = true
	}
	return
}

func main() {

	var wg sync.WaitGroup

	flag.Parse()
	fns := flag.Args()

	qdemolib.SetupRedisForTest(*Cfg)

	client, _ := RedisClient()
	clientListen, _ := RedisClient()

	// ------------------------------------------------------ new -----------------------------------------------------------------------------------------------------
	if *To != "" {
		mm := make(map[string]interface{})
		mm["To"] = *To
		mm["Path"] = *Path
		mm["TrxId"] = *TrxId
		mm["ClientTrxId"] = *ClientTrxId

		dest := ""
		if strings.HasPrefix(*To, "rps://tracer/") {
			dest = "trx:" + *TrxId
			mm["Scheme"] = "rps"
		} else {
			fmt.Printf("Error: -- invalid to -- %s, %s\n", *To, godebug.LF())
		}

		err := client.Cmd("PUBLISH", dest, lib.SVar(mm)).Err
		if err != nil {
			fmt.Printf("Error: %s, %s\n", err, godebug.LF())
		}
	}

	if !*Wait {
		os.Exit(0)
	}

	// ------------------------------------------------------ old -----------------------------------------------------------------------------------------------------
	if false {
		for i := 0; i < len(fns); i += 2 {
			// err := client.Cmd("PUBLISH", "key", "value").Err // iterate over CLI
			if i+1 < len(fns) {
				if *Debug {
					fmt.Printf("PUBLISH %s %s\n", fns[i], fns[i+1])
				}
				err := client.Cmd("PUBLISH", fns[i], fns[i+1]).Err
				if err != nil {
					fmt.Printf("Error: %s\n", err)
				}
			} else {
				fmt.Printf("Usage: should have even number of arguments, found odd, %s skipped\n", fns[i])
			}
		}
	}

	// ---------------------------------------------------------- connectd to redis - do work -------------------------------------------------------------------------
	subClient := pubsub.NewSubClient(clientListen)

	sr := subClient.PSubscribe(*ListenTo)
	if sr.Err != nil {
		fmt.Fprintf(os.Stderr, "%sError: subscribe, %s.%s\n", MiscLib.ColorRed, sr.Err, MiscLib.ColorReset)
	}

	subChan := make(chan *pubsub.SubResp)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			subChan <- subClient.Receive()
		}
	}()

	wg.Add(1)
	go func() {
		// fmt.Printf("AT: %s\n", godebug.LF())
		ticker := time.NewTicker(time.Duration(60) * time.Second)
		defer wg.Done()
		for {
			select {
			case sr = <-subChan:
				if *Debug {
					fmt.Printf("\n***************** Got a message, sr=%+v, %s\n\n", godebug.SVar(sr), godebug.LF())
				}
				var mm map[string]interface{}
				err := json.Unmarshal([]byte(sr.Message), &mm) // 1. Parse sr.Message - if "chat" then
				if err != nil {
					fmt.Fprintf(os.Stderr, "%sError: %s --->>>%s<<<--- AT: %s%s\n", MiscLib.ColorYellow, err, sr.Message, godebug.LF(), MiscLib.ColorReset)
				} else {
					fmt.Printf("Message Is: %s\n", lib.SVarI(mm))
				}
			case <-ticker.C:
				if *Debug {
					fmt.Printf("periodic ticker...\n")
				}
			}
		}
	}()

	// fmt.Printf("AT: %s\n", godebug.LF())
	wg.Wait()

}
