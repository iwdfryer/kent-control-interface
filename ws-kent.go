package main

import (
	"github.com/iwdfryer/kent"
	"github.com/iwdfryer/kent/proto/kentpb"

	"github.com/iwdfryer/utensils/logr"

	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

type bridgeCtx struct {
	wsSrv  *http.ServeMux
	tcpSrv kent.Server
	mutex  sync.Mutex
	cl     *ClientList
}

type wsMsg struct {
	ID     uuid.UUID
	Binary string
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

/**************************************************************
 *                        KENT METHODS                        *
 **************************************************************/

/*
kentMsgHandler - When messages are sent to the kent server we pass this message on
				 to all connected websocket clients.
*/

func (ctx *bridgeCtx) kentMsgHandler(dispenserID uuid.UUID, resp *kentpb.CliToSrv) {

	b, err := proto.Marshal(resp)
	if err != nil {
		fmt.Println("Error marshaling", err)
	}

	payload := wsMsg{
		ID:     dispenserID,
		Binary: base64.StdEncoding.EncodeToString([]byte(b)),
	}
	p, _ := json.Marshal(payload)

	ctx.mutex.Lock()
	for i := range ctx.cl.Clients {
		ctx.cl.Clients[i].Connection.WriteMessage(1, p)
	}
	ctx.mutex.Unlock()
}

func (ctx *bridgeCtx) onKentDispenserOnline(dispenserID uuid.UUID) {
	logr.Infof("Dispenser connected: %s", dispenserID)
}

func (ctx *bridgeCtx) onKentDispenserDisconn(dispenserID uuid.UUID) {
	logr.Infof("Dispenser disconnected: %s", dispenserID)
}

func (ctx *bridgeCtx) kentSubscribe() {
	ctx.tcpSrv.RegisterOnClientOnlineCb(ctx.onKentDispenserOnline)
	ctx.tcpSrv.RegisterOnClientDisconnCb(ctx.onKentDispenserDisconn)

	ctx.tcpSrv.RegisterOnDataCb(&kent.TCPKentServerHdlr{
		&kentpb.CliToSrv_DispenserProcessResp{},
		ctx.kentMsgHandler,
	})
	ctx.tcpSrv.RegisterOnDataCb(&kent.TCPKentServerHdlr{
		&kentpb.CliToSrv_SnapshotRpt{},
		ctx.kentMsgHandler,
	})
	ctx.tcpSrv.RegisterOnDataCb(&kent.TCPKentServerHdlr{
		&kentpb.CliToSrv_DispenserStateRpt{},
		ctx.kentMsgHandler,
	})
	ctx.tcpSrv.RegisterOnDataCb(&kent.TCPKentServerHdlr{
		&kentpb.CliToSrv_LogRpt{},
		ctx.kentMsgHandler,
	})
	ctx.tcpSrv.RegisterOnDataCb(&kent.TCPKentServerHdlr{
		&kentpb.CliToSrv_EepromRRpt{},
		ctx.kentMsgHandler,
	})
	ctx.tcpSrv.RegisterOnDataCb(&kent.TCPKentServerHdlr{
		&kentpb.CliToSrv_DispenserPidDbgRpt{},
		ctx.kentMsgHandler,
	})
	ctx.tcpSrv.RegisterOnDataCb(&kent.TCPKentServerHdlr{
		&kentpb.CliToSrv_DbgScaleReadResp{},
		ctx.kentMsgHandler,
	})
	ctx.tcpSrv.RegisterOnDataCb(&kent.TCPKentServerHdlr{
		&kentpb.CliToSrv_FryerStateRpt{},
		ctx.kentMsgHandler,
	})
	ctx.tcpSrv.RegisterOnDataCb(&kent.TCPKentServerHdlr{
		&kentpb.CliToSrv_FryerUnlockFreezerResponse{},
		ctx.kentMsgHandler,
	})
	ctx.tcpSrv.RegisterOnDataCb(&kent.TCPKentServerHdlr{
		&kentpb.CliToSrv_FryerCookModeResponse{},
		ctx.kentMsgHandler,
	})
	ctx.tcpSrv.RegisterOnDataCb(&kent.TCPKentServerHdlr{
		&kentpb.CliToSrv_FryerHotHoldResponse{},
		ctx.kentMsgHandler,
	})
	ctx.tcpSrv.RegisterOnDataCb(&kent.TCPKentServerHdlr{
		&kentpb.CliToSrv_FryerFreezerResp{},
		ctx.kentMsgHandler,
	})
	ctx.tcpSrv.RegisterOnDataCb(&kent.TCPKentServerHdlr{
		&kentpb.CliToSrv_FryerUnlockFreezerResponse{},
		ctx.kentMsgHandler,
	})
	ctx.tcpSrv.RegisterOnDataCb(&kent.TCPKentServerHdlr{
		&kentpb.CliToSrv_FryerCookModeResponse{},
		ctx.kentMsgHandler,
	})
	ctx.tcpSrv.RegisterOnDataCb(&kent.TCPKentServerHdlr{
		&kentpb.CliToSrv_FryerHotHoldResponse{},
		ctx.kentMsgHandler,
	})
	ctx.tcpSrv.RegisterOnDataCb(&kent.TCPKentServerHdlr{
		&kentpb.CliToSrv_FryerFreezerResp{},
		ctx.kentMsgHandler,
	})
}

/**************************************************************
 *                         WS METHODS                         *
 **************************************************************/

/*
ClientList - The list of clinets currently connected to the websocket server.
*/
type ClientList struct {
	Clients []Client
}

/*
Client - To store each individual client's ID and web socket connection.
*/
type Client struct {
	ID         string
	Connection *websocket.Conn
}

/*
addClient - To add a client to the list when they connect.
*/
func (ctx *bridgeCtx) addClient(client Client) *ClientList {

	ctx.cl.Clients = append(ctx.cl.Clients, client)

	return ctx.cl

}

/*
removeClient - To remove a client from the list when they disconnect.
*/
func (ctx *bridgeCtx) removeClient(client Client) *ClientList {

	for index, cli := range ctx.cl.Clients {

		if cli.ID == client.ID {
			ctx.cl.Clients = append(ctx.cl.Clients[:index], ctx.cl.Clients[index+1:]...)
		}

	}

	return ctx.cl
}

/*
websocketHandler - When connection to the web socket server is made, create client and

	loop waiting for received messages to handle.
*/
func (ctx *bridgeCtx) websocketHandler(w http.ResponseWriter, r *http.Request) {

	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true

	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := Client{
		ID:         uuid.New().String(),
		Connection: conn,
	}

	ctx.addClient(client)
	fmt.Println("New Client is connected, total: ", len(ctx.cl.Clients))

	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			ctx.removeClient(client)
			log.Println("total clients ", len(ctx.cl.Clients))

			return
		}

		ctx.wsMsgHandler(payload)
	}

}

/*
wsMsgHandler  - When messages are sent to the web socket server we pass this message on

	to the tcp connection.
*/
func (ctx *bridgeCtx) wsMsgHandler(payload []byte) {

	msg := wsMsg{}
	err := json.Unmarshal(payload, &msg)
	if err != nil {
		fmt.Println("unmarshalling error. " + err.Error())
		return
	}

	b, err := base64.StdEncoding.DecodeString(msg.Binary)

	req := &kentpb.SrvToCli{}
	err = proto.Unmarshal(b, req)
	if err != nil {
		fmt.Println("Error unmarshaling", err)
	}

	ctx.tcpSrv.SendData(msg.ID, req)
	time.Sleep(100 * time.Millisecond)
	return
}

/**************************************************************
 *                            MAIN                            *
 **************************************************************/

/*
Options:

	[-broker <uri>]             Broker URI
	[-kentIP <uri>]             Kent Server binding IP
	[-kentPort <port>]          Kent Server Port
*/
func main() {
	kentIP := flag.String("kentIP", "0.0.0.0", "The Kent Server IP to bind to ex: 0.0.0.0")
	kentPort := flag.String("kentPort", "64532", "The Kent Server port to listen to. ex: 64532")
	flag.Parse()

	ctx := bridgeCtx{}
	ctx.cl = &ClientList{}

	//kent server
	ctx.tcpSrv = kent.NewKentServer()
	ctx.kentSubscribe()

	err := ctx.tcpSrv.StartServer(*kentIP, *kentPort)
	if err != nil {
		os.Exit(1)
	}

	//web socket server
	ctx.wsSrv = http.NewServeMux()
	ctx.wsSrv.HandleFunc("/ws", ctx.websocketHandler)
	ctx.wsSrv.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static")
	})

	fmt.Println("Server is running: ws://0.0.0.0:3000")
	http.ListenAndServe("0.0.0.0:3000", ctx.wsSrv)
}
