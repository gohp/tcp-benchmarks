package main

import (
	"github.com/gohp/goutils/ws/server"
	"net/http"
)

// ws text msg 规则
func wsRule(msg []byte) []byte {
	// 编写 规则
	return []byte("{\"msg\":\"pong\"}")
}

// http server
func InitHttpServer() {
	http.HandleFunc("/", server.WsHandlerHttp(wsRule))

	if err := http.ListenAndServe(":7778", nil); err != nil {
		panic(err)
	}
}

func main() {
	InitHttpServer()
}