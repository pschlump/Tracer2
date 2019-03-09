package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/pschlump/Tracer2/qdemolib"
	"github.com/pschlump/radix.v2/redis" // Modified pool to have NewAuth for authorized connections
)

var Debug = flag.Bool("debug", false, "Debug flag")                      // 2
var Cfg = flag.String("cfg", "../global_cfg.json", "Configuration file") // 0
func init() {
	flag.BoolVar(Debug, "D", false, "Debug flag")                        // 2
	flag.StringVar(Cfg, "c", "../global_cfg.json", "Configuration file") // 0
}

func main() {

	flag.Parse()
	fns := flag.Args()

	qdemolib.SetupRedisForTest(*Cfg)

	client, err := redis.Dial("tcp", qdemolib.ServerGlobal.RedisConnectHost+":"+qdemolib.ServerGlobal.RedisConnectPort)
	if err != nil {
		log.Fatal(err)
	}
	if qdemolib.ServerGlobal.RedisConnectAuth != "" {
		err = client.Cmd("AUTH", qdemolib.ServerGlobal.RedisConnectAuth).Err
		if err != nil {
			log.Fatal(err)
		} else {
			fmt.Fprintf(os.Stderr, "Success: Connected to redis-server with AUTH.\n")
		}
	} else {
		fmt.Fprintf(os.Stderr, "Success: Connected to redis-server.\n")
	}

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

/* vim: set noai ts=4 sw=4: */
