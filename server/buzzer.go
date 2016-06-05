package main

import (
	"github.com/hybridgroup/gobot"
	"github.com/sparkybots/sparky/server/board"
	"strconv"
)

const (
	BuzzerPlayReq string = "PLAY"
)

type Buzzer struct {
	board     *board.Board
	reqQueue  chan BuzzerReq
	respQueue chan Work
}

type BuzzerReq struct {
	ID      string
	ReqType string
	Result  int
}

func (r BuzzerReq) GetID() string {
	return r.ID
}

func (r BuzzerReq) GetType() string {
	return r.ReqType
}

func (r BuzzerReq) GetRespValue() string {
	return strconv.Itoa(r.Result)
}

func CreateBuzzer(b *board.Board, respQ chan Work) Buzzer {

	buzzer := Buzzer{
		board:     b,
		reqQueue:  make(chan BuzzerReq, 100),
		respQueue: respQ,
	}

	gobot.On(b.Event("BuzzerDone"), buzzer.processBuzzerDone)
	return buzzer
}

func (bz *Buzzer) PlayTone(id string, freq int, delay int) error {
	if delay > 0 {
		req := BuzzerReq{ID: id, ReqType: BuzzerPlayReq, Result: 0}
		bz.reqQueue <- req
	}
	return bz.board.RoverPlayTone(byte(freq), delay)
}

func (bz *Buzzer) BuzzerOff() error {
	return bz.board.RoverBuzzerOff()
}

func (bz *Buzzer) Beep() error {
	return bz.board.RoverBeep()
}

func (bz *Buzzer) processBuzzerDone(data interface{}) {
	req := <-bz.reqQueue
	bz.respQueue <- req
}
