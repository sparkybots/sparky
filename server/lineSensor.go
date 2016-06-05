package main

import (
	"fmt"
	"github.com/hybridgroup/gobot"
	"github.com/sparkybots/sparky/server/board"
	"strconv"
)

const (
	LineReq       string = "LINE"
	LineLeftResp  string = "LINE_LEFT"
	LineRightResp string = "LINE_RIGHT"
)

type LineSensor struct {
	board     *board.Board
	reqQueue  chan LineSensorReq
	respQueue chan Work
}

type LineSensorReq struct {
	ID      string
	ReqType string
	Result  int
}

func (r LineSensorReq) GetID() string {
	return r.ID
}

func (r LineSensorReq) GetType() string {
	return r.ReqType
}

func (r LineSensorReq) GetRespValue() string {
	return strconv.Itoa(r.Result)
}

func CreateLineSensor(b *board.Board, respQ chan Work) LineSensor {

	sensor := LineSensor{
		board:     b,
		reqQueue:  make(chan LineSensorReq, 100),
		respQueue: respQ,
	}

	gobot.On(b.Event("RoverLineResponse"), sensor.processLineResponse)
	return sensor
}

func (l *LineSensor) readLineSensors(id string) error {
	req := LineSensorReq{ID: id, ReqType: LineReq, Result: 0}
	l.reqQueue <- req
	return l.board.RoverReadLineSensors()
}

func (l *LineSensor) processLineResponse(data interface{}) {
	req := <-l.reqQueue
	val := data.(uint8)
	lResp := LineSensorReq{ID: req.GetID(), ReqType: LineLeftResp, Result: int(^val & 0x01)}
	rResp := LineSensorReq{ID: req.GetID(), ReqType: LineRightResp, Result: int(^(val >> 1) & 0x01)}
	l.respQueue <- lResp
	l.respQueue <- rResp
	fmt.Println("LineSensor: Got line response, assigning to ID ", req.ID)
}
