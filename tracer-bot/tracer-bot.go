package main

//
// TODO:
//
// 	1. 	xyzzyReplyTo - should check for a ReplyTo value!
//

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/pschlump/Go-FTL/server/lib"
	"github.com/pschlump/Go-FTL/server/sizlib"
	"github.com/pschlump/MiscLib"
	"github.com/pschlump/Tracer2/lib"
	"github.com/pschlump/Tracer2/qdemolib"
	"github.com/pschlump/godebug"
	"github.com/pschlump/mon-alive/lib"
	"github.com/pschlump/radix.v2/pubsub"
	"github.com/pschlump/radix.v2/redis" // Modified pool to have NewAuth for authorized connections
)

var Debug = flag.Bool("debug", false, "Debug flag")                      // 0
var Debug2 = flag.Bool("db2", false, "Debug flag")                       // .	-- Print out Periodic Ticker info
var Debug3 = flag.Bool("db3", false, "Debug flag")                       // .
var Reload = flag.Bool("Reload", true, "Reload saved state")             // .		// Wisky Tango Foxtrot!
var Cfg = flag.String("cfg", "../global_cfg.json", "Configuration file") // 1
var ListenTo = flag.String("listen", "trx:*", "What to listen to")       // 2
var IsPattern = flag.Bool("pattern", false, "Listen is pattern")         // 3
func init() {
	flag.BoolVar(Debug, "D", false, "Debug flag")                        // 0
	flag.StringVar(Cfg, "c", "../global_cfg.json", "Configuration file") // 1
	flag.StringVar(ListenTo, "l", "trx:*", "What to listen to")          // 2
	flag.BoolVar(IsPattern, "p", false, "Listen is pattern")             // 3
}

type MsgSType struct {
	Msg  string `json:"msg"`
	Name string `json:"name"`
	Body string `json:"body"`
	User string `json:"user"`
}

type MsgType struct {
	GMsg  string   `json:"msg"`
	Id    string   `json:"Id"`
	TrxId string   `json:"TrxId"`
	Orig  MsgSType `json:"body"`
}

func main() {

	var conFlag = false

	flag.Parse()
	fns := flag.Args()
	if len(fns) != 0 {
		flag.Usage()
		os.Exit(1)
	}

	qdemolib.SetupRedisForTest(*Cfg)

	client, conFlag := RedisClient()
	monClient, _ := RedisClient()
	clientListen, _ := RedisClient()

	mon := MonAliveLib.NewMonIt(func() *redis.Client { return monClient }, func(conn *redis.Client) {})
	mon.SendPeriodicIAmAlive("chat-bot")

	var wg sync.WaitGroup

	if conFlag {
		fmt.Fprintf(os.Stderr, "Success: Connected to redis-server.\n")
	} else {
		os.Exit(1)
	}

	// fmt.Printf("AT: %s\n", godebug.LF())
	mt := NewMonitorTrack()

	// If reload of saved state then grab it from Redis
	if *Reload {
		cfg, err := client.Cmd("GET", "trx|MonitorTrack").Str()
		fmt.Printf("Loading Old Config=%s\n", cfg)
		if err == nil {
			err = json.Unmarshal([]byte(cfg), mt) // 1. Parse sr.Message - if "chat" then
			if err != nil {
				fmt.Printf("Failed to load old config, value [%s] did not parse, err=%s\n", cfg, err)
			}
		}
		mt.SendDataToAll()
	}

	// ----------------------------------------------------------------------- connectd to redis - do work -------------------------------------------------------------------------
	subClient := pubsub.NewSubClient(clientListen)

	// sr := subClient.PSubscribe("trx:*")
	sr := subClient.PSubscribe(*ListenTo)
	if sr.Err != nil {
		fmt.Fprintf(os.Stderr, "%sError: subscribe, %s.%s\n", MiscLib.ColorRed, sr.Err, MiscLib.ColorReset)
	}

	if sr.Type != pubsub.Subscribe {
		log.Fatalf("Did not receive a subscribe reply\n")
	}

	if sr.SubCount < 1 {
		log.Fatalf("Unexpected subscription count, Expected: atleast 1, Found: %d\n", sr.SubCount)
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
			// fmt.Printf("Before Receive AT: %s\n", godebug.LF())
			select {
			case sr = <-subChan:
				// sr={"Type":3,"Channel":"listen","Pattern":"","SubCount":0,"Message":"abcF","Err":null}
				if *Debug {
					fmt.Printf("\n***************** Got a message, sr=%+v, %s\n\n", godebug.SVar(sr), godebug.LF())
				}
				var mm map[string]interface{}
				err := json.Unmarshal([]byte(sr.Message), &mm) // 1. Parse sr.Message - if "chat" then
				if err != nil {
					fmt.Printf("Error: %s --->>>%s<<<--- AT: %s\n", err, sr.Message, godebug.LF())
					fmt.Fprintf(os.Stderr, "%sError: %s --->>>%s<<<--- AT: %s%s\n", MiscLib.ColorYellow, err, sr.Message, godebug.LF(), MiscLib.ColorReset)
				} else {
					// 1. 2 different TrxId's at this point
					//		(1) is the Tracer - who to send data back to
					//		(2) is the client - who to listen for
					mm_NsId, err := tracerlib.GetFromMapInterface("NsId", mm)               // ??
					mm_TheKey, err := tracerlib.GetInt64FromMapInterface("maxKey", mm)      // from ?? listen m.Message?
					mm_TrxId, err := tracerlib.GetFromMapInterface("TrxId", mm)             // should originally come from cookie
					mm_ClientTrxId, err := tracerlib.GetFromMapInterface("ClientTrxId", mm) // should originally come from cookie
					mm_Path, err := tracerlib.GetFromMapInterface("Path", mm)               // should originally come from cookie
					mm_Scheme, err := tracerlib.GetFromMapInterface("Scheme", mm)           // should originally come from cookie
					newKey := "r:listen"                                                    // xyzzyReplyTo - should check for a ReplyTo value!
					newMsg := ""

					fmt.Printf("\nmm=%s mm_NsId=%v mm_TheKey=%v mm_TrxId=%v mm_ClientTrxId=%v mm_Path=%s mm_Scheme=%s\n", lib.SVar(mm), mm_NsId, mm_TheKey, mm_TrxId, mm_ClientTrxId, mm_Path, mm_Scheme)

					/*
					   http://dev2.test2.com:16010/ -- The Socket.IO control/display "tracer2" front end.
					   	81f8985d-d125-4480-43c6-b813c0930dff -- Trx-Id

					   http://localhost:16000/api/table/log?id=43 -- the client making the SQL request
					   	bbfd91e7-7e80-46ba-40a9-77a8cf70b938 -- Client-Trx-Id
					*/
					alreadyDidIt := false

					// -- -- Dispatch of actions on this end ----------------------------------------------------------------------------------------------
					if match, plist := DispatchMatch(mm, "rps", "/listen-for", "FilterId", "ClientTrxId"); match {
						// rps://tracer/listen-for?ClientTrxId=XXX&FilterId=MostRecent
						//	FilterId=First,Last,MostRecent,#######
						filterId := plist["FilterId"]
						clientTrxId := plist["ClientTrxId"]
						if filterId == "" { //	If not specified then MostRecent
							plist["FilterId"] = "MostRedent"
							filterId = "MostRedent"
						}
						switch filterId {
						case "First":
							// newMsg = fmt.Sprintf(`{ "To":"sio://server/trx", "TrxId":%q, "ClientTrxId":%q, "body":%q}`, mm_ClientTrxId, clientTrxId, `{"some":"first-data-001"}`)
							_, Data, _ := GetSpecificId(client, CmdFirst)
							newMsg = fmt.Sprintf(`{ "To":"sio://server/trx", "TrxId":%q, "body":%s}`, mm_TrxId, lib.SVar(Data))
							mt.AddUser(mm_TrxId, clientTrxId)
							mt.SetSendDataNow(mm_TrxId, false)
							fmt.Printf("%s, %s\n", mt.Dump(), godebug.LF())
							mt.SaveToRedis(client)
						case "Last":
							_, Data, _ := GetSpecificId(client, CmdLast)
							newMsg = fmt.Sprintf(`{ "To":"sio://server/trx", "TrxId":%q, "body":%s}`, mm_TrxId, lib.SVar(Data))
							mt.AddUser(mm_TrxId, clientTrxId)
							mt.SetSendDataNow(mm_TrxId, false)
							fmt.Printf("%s, %s\n", mt.Dump(), godebug.LF())
							mt.SaveToRedis(client)
						default: // is really get data for Nth
							mm_TheKey, err = strconv.ParseInt(filterId, 10, 64)
							if err != nil {
								fmt.Printf("\n%s Invalid filterId [%s]%s\n", MiscLib.ColorRed, filterId, MiscLib.ColorReset)
								mm_TheKey = 0
							}
							// xyzzy2 - check that front end reaches this with a filterId -and- valid fitlerId
							fallthrough
						case "MostRecent":
							Data := GetDataForId(mm_TheKey, client)
							// - setup for continuing to send on any new data for this ClientTrxId -> mm_TrxId	-- save to config for later matches --
							// - set client can *NOT* accept data at this time
							// - send data to client
							newMsg = fmt.Sprintf(`{ "To":"sio://server/trx", "TrxId":%q, "body":%s}`, mm_TrxId, lib.SVar(Data))
							fmt.Printf("\n%s+===================================================================\n", MiscLib.ColorYellow)
							fmt.Printf("| Got to MostRecent Case!, %s, %s\n", newMsg, godebug.LF())
							fmt.Printf("+===================================================================%s\n", MiscLib.ColorReset)
							mt.AddUser(mm_TrxId, clientTrxId)
							if Data == "" {
								mt.SetSendDataNow(mm_TrxId, true) // data will have just been sent, client must "turn" on flow again
							} else {
								mt.SetSendDataNow(mm_TrxId, false) // data will have just been sent, client must "turn" on flow again
							}
							fmt.Printf("%s, %s\n", mt.Dump(), godebug.LF())
							mt.SaveToRedis(client)

							//default: // is really get data for Nth
							// - check that front end reaches this with a filterId -and- valid fitlerId
							// -check that value is a number!
							// -get data for that number
							//	Data := GetDataForId(mm_TheKey, client)
							//	newMsg = fmt.Sprintf(`{ "To":"sio://server/trx", "TrxId":%q, "body":%s}`, mm_TrxId, lib.SVar(Data))
							// - **trun off** continuing to send on any new data for this ClientTrxId -> mm_TrxId
							//	mt.SetSendDataNow(mm_TrxId, false) // data will have just been sent, client must "turn" on flow again
						}
					} else if match, plist := DispatchMatch(mm, "rps", "/end-listen-for", "ClientTrxId"); match {
						// **trun off** continuing to send on any new data for this ClientTrxId -> mm_TrxId
						mt.DeleteListenFor(mm_TrxId, plist["ClientTrxId"])
						mt.SaveToRedis(client)
					} else if match, plist := DispatchMatch(mm, "rps", "/ready-for-more-data", "ClientTrxId"); match {
						// Switch flag in config - client can accept data
						if cti, ok := plist["ClientTrxId"]; ok && cti != "" {
							mt.AddUser(mm_TrxId, plist["ClientTrxId"])
						}
						mt.SetSendDataNow(mm_TrxId, true) // data will have just been sent, client must "turn" on flow again
						fmt.Printf("%sGot /ready-for-more-data\nSetting %s to true\n%s, %s%s\n", MiscLib.ColorCyan, mm_TrxId, mt.Dump(), godebug.LF(), MiscLib.ColorReset)
						mt.SaveToRedis(client)
					} else if match, _ := DispatchMatch(mm, "rps", "/list-trx", "x"); match { // send back filter info to client
						newMsg = fmt.Sprintf(`{ "To":"sio://server/trx", "TrxId":%q, "body":%q}`, mm_TrxId, mt.ListTrx())
					} else if match, _ := DispatchMatch(mm, "rps", "/dump", "x"); match { // send back filter info to client
						newMsg = fmt.Sprintf(`{ "To":"sio://server/trx", "TrxId":%q, "body":%q}`, mm_TrxId, `{"status":"success","body":"/dump received"}`)
					} else if match, _ := DispatchMatch(mm, "rps", "/ping", "x"); match { // for check that I am alive
						newMsg = fmt.Sprintf(`{ "To":"sio://server/trx", "TrxId":%q, "body":%q}`, mm_TrxId, `{"status":"success","body":"/ping received"}`)
						// report that I am alive - and that circuit is complete -
					} else if match, _ := DispatchMatch(mm, "rps", "/uri-start", "ClientTrxId"); match { // for check that I am alive
						// Hay Babby -- xyzzy - this is the one where we receive new data on uri-start from tr/trace.go package
						// watch this for timeout - Yellow/Red - error timeout - 2s - Yellow, 30s - Red
						newMsg = fmt.Sprintf(`{ "To":"sio://server/trx", "TrxId":%q, "body":%q}`, mm_TrxId, `{"status":"success"}`) // report that I am alive
						mt.AddUser("xyzzyTrxId", mm_ClientTrxId)
						// xyzzy - should set timeout threshold of X seconds - if not cleard then send error to human-user
					} else if match, _ := DispatchMatch(mm, "rps", "/uri-end", "ClientTrxId"); match { // for check that I am alive
						// Hay Babby -- xyzzy - this is the one where we got the full load! -- Send to client.
						newMsg = fmt.Sprintf(`{ "To":"sio://server/trx", "TrxId":%q, "body":%q}`, mm_TrxId, `{"status":"success","at":"/uri-end"}`)
						// --------------------------------------------------------------- send -------------------------------------------------------------------
						fmt.Printf("%s%s, Before! Match on %s\n %s%s\n", MiscLib.ColorMagenta, mm_ClientTrxId, mt.Dump(), godebug.LF(), MiscLib.ColorReset)
						doSend, ToTrxIds := mt.IsMatch(mm_ClientTrxId) // - 1 - identify listeners for specific TrxId's - send to them
						fmt.Printf("%s/uri-end received, doSend=%v ToTrxIds=%s, %s%s\n", MiscLib.ColorGreen, doSend, lib.SVar(ToTrxIds), godebug.LF(), MiscLib.ColorReset)
						alreadyDidIt = true
						if doSend {
							Data := GetDataForId(mm_TheKey, client)
							for _, aTrxId := range ToTrxIds {
								fmt.Printf("\taTrxId=%s At:%s\n", aTrxId, godebug.LF())
								// newMsg = fmt.Sprintf(`{ "To":"sio://server/trx", "TrxId":%q, "body":%q}`, aTrxId, `{"status":"success","at":"/uri-end"}`)
								newMsg = fmt.Sprintf(`{ "To":"sio://server/trx", "TrxId":%q, "body":%s}`, aTrxId, lib.SVar(Data))
								if *Debug {
									fmt.Printf("PUBLISH \"%s\" \"%s\"\n", newKey, newMsg) // xyzzy - logrus loggin of message - or debug at this point
								}
								err = client.Cmd("PUBLISH", newKey, newMsg).Err
								if err != nil {
									fmt.Printf("Error: %s\n", err)
									fmt.Fprintf(os.Stderr, "Error: %s\n", err)
								}
								mt.SetSendDataNow(aTrxId, false) // data will have just been sent, client must "turn" on flow again
							}
						}
					} else {
						newMsg = fmt.Sprintf(`{ "To":"sio://server/trx", "TrxId":%q, "body":%q}`, mm_TrxId, `{"status":"ok"}`)
					}
					// -- -- end --------------------------------------------------------------------------------------------------------------------------

					if !alreadyDidIt {
						if *Debug {
							fmt.Printf("PUBLISH \"%s\" \"%s\"\n", newKey, newMsg) // xyzzy - logrus loggin of message - or debug at this point
						}
						err = client.Cmd("PUBLISH", newKey, newMsg).Err
						if err != nil {
							fmt.Printf("Error: %s\n", err)
							fmt.Fprintf(os.Stderr, "Error: %s\n", err)
						}
					}
				}
			case <-ticker.C:
				if *Debug2 {
					fmt.Printf("periodic ticker...\n")
				}
				// xyzzy - set of at one sec - check for Yellow-Red errors - started but not completed transactions to server
			}
			// fmt.Printf("After Receive AT: %s\n", godebug.LF())
		}
	}()

	// fmt.Printf("AT: %s\n", godebug.LF())
	wg.Wait()
}

// If you do not have it then get it
// If it is dirty then get it
// Listten for updates / add / remove from list
// http://redis.io/commands/SMEMBERS
//	SADD, SREM
func GetTrxList(client *redis.Client) (rv []string) {
	ls, err := client.Cmd("SMEMBERS", "trx:list").List()
	fmt.Printf("xx=%s\n", godebug.SVarI(ls))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting back trx:list, %s\n", err)
		return
	}
	rv = ls
	return
}

// Need paramName to be struct/type - with name, defaultValue, reMatchToValidate, requiredFlag
// if match, plist := DispatchMatch ( mm, "rps", "/listen-for", "FilterId", "ClientTrxId" ) ; match {
func DispatchMatch(mm map[string]interface{}, scheme string, path string, paramNames ...string) (match bool, plist map[string]string) {
	get := func(name string) (rv string) {
		t1, ok := mm[name]
		if ok {
			t2, ok := t1.(string)
			if ok {
				rv = t2
			}
		}
		return
	}
	s := get("Scheme")
	p := get("Path")
	if scheme == s && path == p {
		match = true
		// extract paramNames into plist
		plist = make(map[string]string)
		for _, vv := range paramNames {
			plist[vv] = get(vv)
		}
	}
	return
}

// get last ID
// Id, Seq := GetLastId()
func GetLastId(conn *redis.Client) (Id, Seq string) {
	return
}

// get data for that ID number number
func GetDataForId(theKey int64, conn *redis.Client) (Data string) {
	//t, err := strconv.ParseInt(theKey, 10, 64)
	//if err != nil {
	//	fmt.Printf("Error(11006) error getting key, should be int, got:%s for %s, err=%s\n", ss, theKey, err)
	//	return
	//}
	fmt.Printf("GetDataForId(theKey=%d), %s\n", theKey, godebug.LF())
	s, err := GetOutput(theKey, conn)
	if err == nil {
		Data = s
	}
	return
}

type MonitorTrackUser struct {
	TrxId        string    // TrxId of the client that wants to do Tracer2 - this is the output display
	ClientTrxId  []string  // set of TrxIds that are being activly traced
	MinSeqNo     int64     // Lowest confirmed SeqNo
	MaxSeqNo     int64     // Hightest observed (from a Pub/Sub) SeqNo but may not have been confirmed
	SendDataNow  bool      // flag, if true then client is ready to accept data
	ExpiresAfter time.Time // Requries revalidation /ping -> /alive for client or discard if no /Alive message
}

type MonitorTrack struct {
	UsersWatching map[string]MonitorTrackUser
	mutex         sync.RWMutex //
}

/*
hh.mutex.Lock()
hh.mutex.Unlock()
hdlr.mutex.RLock()
hdlr.mutex.RUnlock()
*/

func NewMonitorTrack() (mt *MonitorTrack) {
	return &MonitorTrack{
		UsersWatching: make(map[string]MonitorTrackUser),
	}
}

func (mt *MonitorTrack) Dump() (rv string) {
	rv = `{ "UsersWatching": [` + "\n"
	com := " "
	for ii, vv := range mt.UsersWatching {
		rv += fmt.Sprintf(`    %s { "Key":%q, "TrxId":%q, "ClientTrxId":%q, "SendDataNow":%v }`+"\n", com, ii, vv.TrxId, vv.ClientTrxId, vv.SendDataNow)
		com = ","
	}
	rv += "] }\n"
	return
}

func (mt *MonitorTrack) ListTrx() (rv string) {
	mm := make(map[string]bool)
	for _, vv := range mt.UsersWatching {
		for _, ww := range vv.ClientTrxId {
			mm[ww] = true
		}
	}
	rv = `{ "AvailTrx": [` + "\n"
	com := " "
	for vv := range mm {
		rv += fmt.Sprintf(`    %s { "TrxId":%q }`+"\n", com, vv)
		com = ","
	}
	rv += "] }\n"
	return
}

func (mt *MonitorTrack) AddUser(TrxId, ClientTrxId string) {

	if TrxId == "xyzzyTrxId" { // xyzzy - Clean this up
		return
	}

	mt.mutex.RLock()
	uu, ok := mt.UsersWatching[TrxId]
	mt.mutex.RUnlock()
	if !ok {
		mt.mutex.Lock()
		if ClientTrxId != "" {
			expireTime := time.Now().Add((24 * 60 * 60) * time.Second) // good for 1 day
			mt.UsersWatching[TrxId] = MonitorTrackUser{
				TrxId:        TrxId,
				ClientTrxId:  []string{ClientTrxId},
				SendDataNow:  true,
				ExpiresAfter: expireTime,
			}
		} else {
			mt.UsersWatching[TrxId] = MonitorTrackUser{
				TrxId: TrxId,
			}
		}
		mt.mutex.Unlock()
	} else {
		if !lib.InArray(ClientTrxId, uu.ClientTrxId) {
			uu.ClientTrxId = append(uu.ClientTrxId, ClientTrxId)
			expireTime := time.Now().Add((24 * 60 * 60) * time.Second) // good for 1 day
			uu.ExpiresAfter = expireTime                               // Update to 1 more day
			mt.mutex.Lock()
			mt.UsersWatching[TrxId] = uu
			mt.mutex.Unlock()
		}
	}

}

// mt.DeleteListenFor(mm_TrxId, plist["ClientTrxId"])
func (mt *MonitorTrack) DeleteListenFor(TrxId, ClientTrxId string) {
	mt.mutex.RLock()
	uu, ok := mt.UsersWatching[TrxId]
	mt.mutex.RUnlock()
	if !ok {
		return
	}

	if nth := lib.InArrayN(ClientTrxId, uu.ClientTrxId); nth >= 0 {
		uu.ClientTrxId = lib.DelFromSliceString(uu.ClientTrxId, nth)
		mt.mutex.Lock()
		mt.UsersWatching[TrxId] = uu
		mt.mutex.Unlock()
	}
}

func (mt *MonitorTrack) IsMatch(ClientTrxId string) (rv bool, SetTrxId []string) {
	fmt.Printf("\tClientTrxId [%s], AT:%s\n", ClientTrxId, godebug.LF())
	if ClientTrxId == "" {
		fmt.Printf("%sClientTrxId is empty!!!%s\n", MiscLib.ColorRed, MiscLib.ColorReset)
		return
	}
	mt.mutex.RLock()
	for TrxId, uu := range mt.UsersWatching {
		fmt.Printf("\tTrxId [%s], Match To ClientTrxIds=%s AT:%s\n", TrxId, lib.SVar(uu.ClientTrxId), godebug.LF())
		if TrxId == "xyzzyTrxId" { // xyzzy - Clean this up
		} else if lib.InArray(ClientTrxId, uu.ClientTrxId) {
			fmt.Printf("\tMatch found, checking SendDataNow\n")
			if uu.SendDataNow {
				fmt.Printf("\tMatch found, checking SendDataNow is true, append\n")
				SetTrxId = append(SetTrxId, TrxId)
			}
		}
	}
	mt.mutex.RUnlock()
	rv = len(SetTrxId) > 0
	return
}

func (mt *MonitorTrack) SetSendDataNow(TrxId string, state bool) {

	mt.mutex.RLock()
	uu, ok := mt.UsersWatching[TrxId]
	mt.mutex.RUnlock()
	if !ok {
		return
	}

	uu.SendDataNow = state

	mt.mutex.Lock()
	mt.UsersWatching[TrxId] = uu
	mt.mutex.Unlock()
}

func (mt *MonitorTrack) SaveToRedis(client *redis.Client) {
	err := client.Cmd("SET", "trx|MonitorTrack", lib.SVar(mt)).Err
	if err != nil {
		fmt.Printf("Error saving MonitorTrack, %s\n", err)
	}
}

func (mt *MonitorTrack) SendDataToAll() {
	mt.mutex.Lock()
	for TrxId, uu := range mt.UsersWatching {
		uu.SendDataNow = true
		mt.UsersWatching[TrxId] = uu
	}
	mt.mutex.Unlock()
}

// xyzzy  -get first ID

// xyzzy - real dispatcher - table of values - match/lookup - save fucntion - execute - with NOT FOUND function

// Modified pool to have NewAuth for authorized connections

// This returns the list of KEYS that can be fetched for details for the Output Page
//
// req is true if this is from a user request, false if from a pubsub/nofitifation
// from a server.
//
// theKey == 141 => 000141 -- from Listen Message
func GetOutput(theKey int64, conn *redis.Client) (string, error) {

	key := fmt.Sprintf(`trx:%06d`, theKey)
	s, err := conn.Cmd("GET", key).Str()

	fmt.Printf("GetOutput(theKey=%d), key=[%s] s=%s, err=%s, %s\n", theKey, key, s, err, godebug.LF())

	var k string
	if err == nil {
		fmt.Printf("GetOutput: for key ->%s<- got ->%s<-, %s\n", key, s, godebug.LF())
		rv, err := conn.Cmd("KEYS", "trx:*").List()
		if err != nil {
			k = "[]"
		} else {
			k = sizlib.SVar(rv)
			if k == "" {
				k = "[]"
			}
		}
		rv2 := fmt.Sprintf(`{"Data":%s,"Keys":%s}`, s, k)
		return rv2, nil
	}
	return fmt.Sprintf(`{"Error":"%s"}`, err), tracerlib.ErrNoOutputFound
}

type CmdFirstLast int

const (
	CmdFirst CmdFirstLast = 1
	CmdLast  CmdFirstLast = 2
)

func GetSpecificId(conn *redis.Client, firstLast CmdFirstLast) (theKey int64, s string, err error) {
	var keys string
	kk, err := conn.Cmd("KEYS", "trx:*").List()
	if err != nil {
		keys = "[]"
	} else {
		theKey = 0
		for _, vv := range kk {
			t, err := strconv.ParseInt(vv, 10, 64)
			if err != nil {
				fmt.Printf("Error(11006) error getting key, should be int, got:%s for %s, err=%s\n", vv, "trx:*", err)
			} else {
				if theKey == 0 {
					theKey = t
				} else if firstLast == CmdFirst && t < theKey {
					theKey = t
				} else if firstLast == CmdLast && t > theKey {
					theKey = t
				}
			}
		}
		keys = sizlib.SVar(kk)
		if keys == "" {
			keys = "[]"
		}
	}

	key := fmt.Sprintf(`trx:%06d`, theKey)
	fmt.Printf("GetFirstId(theKey=%d), key=[%s] s=%s, err=%s, %s\n", theKey, key, s, err, godebug.LF())
	s, err = conn.Cmd("GET", key).Str()
	if err != nil {
		s = "{}"
		err = nil
	}
	s = fmt.Sprintf(`{"Data":%s,"Keys":%s}`, s, keys)
	return
}

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

/* vim: set noai ts=4 sw=4: */
