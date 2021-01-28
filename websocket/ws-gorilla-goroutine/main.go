package main

import (
	"errors"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
	"time"
)

var (
	upGrader websocket.Upgrader
)

type wsConn struct {
	lock                  sync.Mutex
	connId                uint64
	wsSocket              *websocket.Conn
	inChan                chan *WsMessage
	outChan               chan *WsMessage
	closeChan             chan byte
	isClosed              bool
}

type WsMessage struct {
	MsgType int
	MsgData []byte
}

// InitWsConn init a websocket connection
func InitWsConn(wsSocket *websocket.Conn, heartbeatTimeDuration time.Duration) *wsConn {
	w := &wsConn{
		wsSocket:              wsSocket,
		inChan:                make(chan *WsMessage, 1000), // TODO
		outChan:               make(chan *WsMessage, 1000),
		closeChan:             make(chan byte),
	}

	go w.readLoop()
	go w.writeLoop()
	return w
}

// readLoop
func (w *wsConn) readLoop() {
	var (
		msgType int
		msgData []byte
		message *WsMessage
		err     error
	)
	for {
		if msgType, msgData, err = w.wsSocket.ReadMessage(); err != nil {
			w.Close()
			break
		}

		message = &WsMessage{MsgData: msgData, MsgType: msgType}
		select {
		case w.inChan <- message:
		case <-w.closeChan:
			break
		}
	}
}

// writeLoop
func (w *wsConn) writeLoop() {
	var (
		message *WsMessage
		err     error
	)
	for {
		select {
		case message = <-w.outChan:
			if err = w.wsSocket.WriteMessage(message.MsgType, message.MsgData); err != nil {
				w.Close()
				break
			}
		case <-w.closeChan:
			return
		}
	}
}

// readMsg
func (w *wsConn) readMsg() (msg *WsMessage, err error) {
	select {
	case msg = <-w.inChan:
	case <-w.closeChan:
		err = errors.New("Connection Loss")
	}
	return
}

// sendMsg
func (w *wsConn) sendMsg(msg *WsMessage) (err error) {
	select {
	case w.outChan <- msg:
	case <-w.closeChan:
		err = errors.New("Connection Loss")
	default: // 写操作不会阻塞, 因为channel已经预留给websocket一定的缓冲空间
		err = errors.New("Connection full")
	}
	return
}

// Close
func (w *wsConn) Close() {
	w.wsSocket.Close()
	w.lock.Lock()
	defer w.lock.Unlock()

	if !w.isClosed {
		w.isClosed = true
		close(w.closeChan)
	}
}

func InitWsUpGrader() websocket.Upgrader {
	var upGrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// allow CORS
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	return upGrader
}

// http server
func InitHttpServer() {
	upGrader = InitWsUpGrader()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var (
			err     error
			resp    *WsMessage
		)

		wsSocket, err := upGrader.Upgrade(w, r, nil)
		if err != nil {
			panic(err)
		}

		wsConn := InitWsConn(wsSocket, time.Second*60)

		for {
			if _, err = wsConn.readMsg(); err != nil {
				wsConn.Close()
				break
			}

			resp = &WsMessage{MsgType: websocket.TextMessage, MsgData: []byte("{\"msg\":\"pong\"}")}

			if err = wsConn.sendMsg(resp); err != nil {
				wsConn.Close()
				break
			}
		}
	})

	if err := http.ListenAndServe(":7778", nil); err != nil {
		panic(err)
	}
}

func main() {
	InitHttpServer()
}