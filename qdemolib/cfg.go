package qdemolib

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/pschlump/Go-FTL/server/sizlib"
	"github.com/pschlump/MiscLib"
	"github.com/pschlump/radix.v2/pool"
)

// ---- ServerGlobalConfigType -------------------------------------------------------------------------------------------------
type ServerGlobalConfigType struct {
	RedisConnectHost string     `json: "RedisConnectHost"` // Connection infor for Redis Database
	RedisConnectPort string     `json: "RedisConnectPort"` //
	RedisConnectAuth string     `json: "RedisConnectAuth"` //
	RedisPool        *pool.Pool `json: "-"`                // Pooled Redis Client connectioninformation
	mutex            sync.Mutex `json: "-"`                // Lock for redis
}

var ServerGlobal *ServerGlobalConfigType

func NewServerGlobalConfigType() *ServerGlobalConfigType {
	return &ServerGlobalConfigType{
		RedisConnectHost: "127.0.0.1",
		RedisConnectPort: "6379",
		RedisConnectAuth: "",
	}
}

func SetupRedisForTest(redis_cfg_file string) bool {

	if ServerGlobal == nil {
		ServerGlobal = NewServerGlobalConfigType()
	}

	s, err := sizlib.ReadJSONDataWithComments(redis_cfg_file)
	if err != nil {
		log.Fatalf("Unable to read global_cfg.json file Error: %s\n", err)
		return false
	}

	err = json.Unmarshal(s, &ServerGlobal)
	if err != nil {
		log.Fatalf("Unable to parse global_cfg.json file Error: %s\n", err)
		return false
	}

	return ServerGlobal.ConnectToRedis()
}

// ----------------------------------------------------------------------------------------------------------------------------
func (hdlr *ServerGlobalConfigType) ConnectToRedis() bool {
	// Note: best test for this is in the TabServer2 - test 0001 - checks that this works.
	var err error

	hdlr.mutex.Lock()
	defer hdlr.mutex.Unlock()

	dflt := func(a string, d string) (rv string) {
		rv = a
		if rv == "" {
			rv = d
		}
		return
	}

	redis_host := dflt(hdlr.RedisConnectHost, "127.0.0.1")
	redis_port := dflt(hdlr.RedisConnectPort, "6379")
	redis_auth := hdlr.RedisConnectAuth

	if false {
		if redis_auth == "" { // If Redis AUTH section
			hdlr.RedisPool, err = pool.New("tcp", redis_host+":"+redis_port, 20)
		} else {
			hdlr.RedisPool, err = pool.NewAuth("tcp", redis_host+":"+redis_port, 20, redis_auth)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError: Failed to connect to redis-server.%s\n", MiscLib.ColorRed, MiscLib.ColorReset)
			fmt.Printf("Error: Failed to connect to redis-server.\n")
			logrus.Fatalf("Error: Failed to connect to redis-server.\n")
			return false
		} else {
			if db11 {
				fmt.Fprintf(os.Stderr, "Success: Connected to redis-server.\n")
			}
		}
	}

	return true
}

const db11 = true

/* vim: set noai ts=4 sw=4: */
