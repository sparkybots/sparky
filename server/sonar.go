package main

import (
	"fmt"
	"github.com/hybridgroup/gobot"
	"github.com/sparkybots/sparky/server/board"
	"strconv"
)

const (
	MAX_DISTANCE  int    = 9999
	SonarRangeReq string = "RANGE"
	SonarTurnReq  string = "TURN"
)

type Sonar struct {
	board         *board.Board
	rangeReqQueue chan SonarReq
	turnReqQueue  chan SonarReq
	respQueue     chan Work
}

// SonaReq implements the Work interface
type SonarReq struct {
	ID      string
	reqType string
	Result  int
}

func (r SonarReq) GetID() string {
	return r.ID
}

func (r SonarReq) GetType() string {
	return r.reqType
}

func (r SonarReq) GetRespValue() string {
	return strconv.Itoa(r.Result)
}

func CreateSonar(b *board.Board, respQ chan Work) Sonar {

	sonar := Sonar{
		board:         b,
		rangeReqQueue: make(chan SonarReq, 100),
		turnReqQueue:  make(chan SonarReq, 100),
		respQueue:     respQ,
	}

	gobot.On(b.Event("SonarResponse"), sonar.processRangeResponse)
	gobot.On(b.Event("SonarTurnDone"), sonar.processTurnDone)
	return sonar
}

func (s *Sonar) ReadRange(id string) error {
	req := SonarReq{ID: id, reqType: SonarRangeReq, Result: MAX_DISTANCE}
	if err := s.board.RoverSonarRead(); err != nil {
		s.respQueue <- req
		return fmt.Errorf("Error sending read sonar request to board id %s err - %s ", id, err)
	} else {
		fmt.Println("Sonar: sent read request id : ", id)
		s.rangeReqQueue <- req
		return nil
	}
}

func (s *Sonar) processRangeResponse(data interface{}) {

	req := <-s.rangeReqQueue
	req.Result = int(data.(uint8))
	s.respQueue <- req

	fmt.Println("Sonar: Got response ", req.Result, " cm, assigning to ID ", req.ID)
}

func (s *Sonar) Turn(id string, direction string, angle int) error {
	var dir byte
	if direction == "right" {
		dir = board.TurnRight
	} else {
		dir = board.TurnLeft
	}
	angle = angle % 91

	req := SonarReq{ID: id, reqType: SonarTurnReq, Result: 0}
	if err := s.board.RoverSonarTurn(dir, angle); err != nil {
		s.respQueue <- req
		return fmt.Errorf("Error sending turn sonar request to board err - %s ", err)
	} else {
		fmt.Println("Sonar: sent turn request : ", direction, angle)
		s.turnReqQueue <- req
		return nil
	}
}

func (s *Sonar) processTurnDone(data interface{}) {
	req := <-s.turnReqQueue
	s.respQueue <- req

	fmt.Println("Sonar: Got turn response, assigning to ID ", req.ID)
}
