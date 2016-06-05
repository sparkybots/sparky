package main

import (
	"fmt"
	"github.com/hybridgroup/gobot"
	"github.com/sparkybots/sparky/server/board"
	"strconv"
)

const (
	WheelsTurnReq string = "TURN"
	WheelsStepReq string = "STEP"
)

type Wheels struct {
	board        *board.Board
	turnReqQueue chan WheelsReq
	stepReqQueue chan WheelsReq
	respQueue    chan Work
}

type WheelsReq struct {
	ID      string
	ReqType string
	Result  int
}

func (r WheelsReq) GetID() string {
	return r.ID
}

func (r WheelsReq) GetType() string {
	return r.ReqType
}

func (r WheelsReq) GetRespValue() string {
	return strconv.Itoa(r.Result)
}

func CreateWheels(b *board.Board, respQ chan Work) Wheels {

	Wheels := Wheels{
		board:        b,
		turnReqQueue: make(chan WheelsReq, 100),
		stepReqQueue: make(chan WheelsReq, 100),
		respQueue:    respQ,
	}

	gobot.On(b.Event("RoverTurnDone"), Wheels.processTurnDone)
	gobot.On(b.Event("RoverStepDone"), Wheels.processStepDone)
	return Wheels
}

func (wh *Wheels) Turn(id string, direction string, angle int, steps int) (err error) {

	req := WheelsReq{ID: id, ReqType: WheelsTurnReq, Result: 0}

	if direction == "left" {
		err = wh.board.RoverTurn(board.TurnLeft, board.MoveDirFwd, byte(angle), steps)
	} else {
		err = wh.board.RoverTurn(board.TurnRight, board.MoveDirFwd, byte(angle), steps)
	}

	if err == nil {
		fmt.Println("Wheels: sent turn request id : ", id)
		wh.turnReqQueue <- req
	} else {
		wh.respQueue <- req
		err = fmt.Errorf("Error sending turn request to board id %s err - %s ", id, err)
	}

	return
}

func (wh *Wheels) ReverseTurn(id string, direction string, angle int, steps int) (err error) {

	req := WheelsReq{ID: id, ReqType: WheelsTurnReq, Result: 0}

	if direction == "left" {
		err = wh.board.RoverTurn(board.TurnLeft, board.MoveDirRev, byte(angle), steps)
	} else {
		err = wh.board.RoverTurn(board.TurnRight, board.MoveDirRev, byte(angle), steps)
	}

	if err == nil {
		fmt.Println("Wheels: sent turn request id : ", id)
		wh.turnReqQueue <- req
	} else {
		wh.respQueue <- req
		err = fmt.Errorf("Error sending turn request to board id %s err - %s ", id, err)
	}

	return
}

func (wh *Wheels) processTurnDone(data interface{}) {
	req := <-wh.turnReqQueue
	wh.respQueue <- req
}

func (wh *Wheels) Step(id string, direction string, steps int) (err error) {

	req := WheelsReq{ID: id, ReqType: WheelsStepReq, Result: 0}

	if direction == "forward" {
		err = wh.board.RoverStep(board.MoveDirFwd, steps)
	} else {
		err = wh.board.RoverStep(board.MoveDirRev, steps)
	}

	if err == nil {
		fmt.Println("Wheels: sent step request id : ", id)
		wh.stepReqQueue <- req
	} else {
		wh.respQueue <- req
		err = fmt.Errorf("Error sending step request to board id %s err - %s ", id, err)
	}

	return
}

func (wh *Wheels) WheelStep(id string, which string, direction string, steps int) (err error) {

	req := WheelsReq{ID: id, ReqType: WheelsStepReq, Result: 0}

	switch which {
	case "right":
		if direction == "forward" {
			err = wh.board.RoverWheelStep(board.MoveStepRight, board.MoveDirFwd, steps)
		} else {
			err = wh.board.RoverWheelStep(board.MoveStepRight, board.MoveDirRev, steps)
		}
	case "left":
		if direction == "forward" {
			err = wh.board.RoverWheelStep(board.MoveStepLeft, board.MoveDirFwd, steps)
		} else {
			err = wh.board.RoverWheelStep(board.MoveStepLeft, board.MoveDirRev, steps)
		}
	}

	if err == nil {
		fmt.Println("Wheels: sent step request id : ", id)
		wh.stepReqQueue <- req
	} else {
		wh.respQueue <- req
		err = fmt.Errorf("Error sending step request to board id %s err - %s ", id, err)
	}

	return
}

func (wh *Wheels) processStepDone(data interface{}) {
	req := <-wh.stepReqQueue
	wh.respQueue <- req
}

func (wh *Wheels) Run(direction string, leftSpeed int, rightSpeed int) error {
	if direction == "forward" {
		return wh.board.RoverRun(board.MoveDirFwd, byte(leftSpeed), byte(rightSpeed))
	} else {
		return wh.board.RoverRun(board.MoveDirRev, byte(leftSpeed), byte(rightSpeed))
	}
}

func (wh *Wheels) Stop() error {
	return wh.board.RoverStop()
}
