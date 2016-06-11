package qdemolib

import "time"

// Modified pool to have NewAuth for authorized connections

func sendIAmAlive() {
}

func Periodic() {
	// Send periodic I Am Alive Notices -------------------------------------------------------------------
	ticker := time.NewTicker(3 * time.Minute)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				sendIAmAlive()
				//s := ""
				//if s, ok := GlobalCfg["I_Am_Alive_URL"]; ok {
				//	s = doGet(&client, s)
				//} else {
				//	// ,"monitor_url":"http://localhost:8090"
				//	// if s, ok := GlobalCfg["redis_port"]; ok {
				//	s = doGet(&client, GlobalCfg["monitor_url"]+"/api/ping_i_am_alive?item=socket-app:8094&note=Ok")
				//}
				//fmt.Printf("socket-app: %s\n", s)
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

/* vim: set noai ts=4 sw=4: */
