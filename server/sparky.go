package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

var comPort string
var rover Rover
var workResponseQueue = make(chan Work, 100)
var pendingReqs = make(map[string]string)
var lastCmdTime time.Time
var lastPendingTime time.Time

func HandlePoll(w http.ResponseWriter, r *http.Request) {

	if !rover.Connected() {
		if err := rover.Setup(comPort, workResponseQueue); err == nil {
			for key := range pendingReqs {
				delete(pendingReqs, key)
			}
		}
	}

	if !rover.Connected() {
		fmt.Fprintln(w, "_problem Roverduino is not connected")
	} else {
		done := false
		for !done {
			select {
			case resp := <-workResponseQueue:
				delete(pendingReqs, resp.GetID())

				switch resp.GetType() {
				case SonarRangeReq:
					fmt.Println("Sonar range ID: ", resp.GetID(), " value: ", resp.GetRespValue())
					fmt.Fprintf(w, "sonarRange %s\n", resp.GetRespValue())
				case LineLeftResp:
					fmt.Println("Line Left ID: ", resp.GetID(), " value: ", resp.GetRespValue())
					fmt.Fprintf(w, "lineLeft %s\n", resp.GetRespValue())
				case LineRightResp:
					fmt.Println("Line Right ID: ", resp.GetID(), " value: ", resp.GetRespValue())
					fmt.Fprintf(w, "lineRight %s\n", resp.GetRespValue())
				default:
					break
				}

			case <-time.After(5 * time.Millisecond):
				done = true
			}
		}
		pending := ""
		for key, _ := range pendingReqs {
			pending = pending + " " + key
		}
		if pending != "" {
			fmt.Fprintln(w, "_busy "+pending)
			if time.Since(lastPendingTime) >= 5*time.Second {
				if time.Since(lastCmdTime) >= time.Second {
					fmt.Println("Rover - HeartBeat")
					lastCmdTime = time.Now()
					rover.heartBeat()
				}
			}
		} else {
			lastPendingTime = time.Now()
			if time.Since(lastCmdTime) >= time.Second {
				fmt.Println("Rover - HeartBeat")
				lastCmdTime = time.Now()
				rover.heartBeat()
			}
		}
	}
}

func invokeHaandler(w http.ResponseWriter, handler func(map[string]string) error, vars map[string]string) error {
	if !rover.Connected() {
		fmt.Fprintln(w, "_problem Roverduino is not connected")
		return fmt.Errorf("Rover not connected")
	}

	lastCmdTime = time.Now()

	id := vars["id"]
	if err := handler(vars); err != nil {
		fmt.Fprintln(w, "_problem Could not execute command")
		return fmt.Errorf("Could not execute command")
	} else if id != "" {
		pendingReqs[id] = id
		lastPendingTime = time.Now()
	}
	return nil
}

func HandleReset(w http.ResponseWriter, r *http.Request) {
	invokeHaandler(w, rover.Reset, mux.Vars(r))
}

func HandleReadSonar(w http.ResponseWriter, r *http.Request) {
	invokeHaandler(w, rover.ReadSonar, mux.Vars(r))
}

func HandleTurnSonar(w http.ResponseWriter, r *http.Request) {
	invokeHaandler(w, rover.TurnSonar, mux.Vars(r))
}

func HandleCenterSonar(w http.ResponseWriter, r *http.Request) {
	invokeHaandler(w, rover.CenterSonar, mux.Vars(r))
}

func HandleRun(w http.ResponseWriter, r *http.Request) {
	invokeHaandler(w, rover.Run, mux.Vars(r))
}

func HandleStop(w http.ResponseWriter, r *http.Request) {
	invokeHaandler(w, rover.Stop, mux.Vars(r))
}

func HandleTurnCalibrate(w http.ResponseWriter, r *http.Request) {
	invokeHaandler(w, rover.TurnCalibrate, mux.Vars(r))
}

func HandleTurn(w http.ResponseWriter, r *http.Request) {
	invokeHaandler(w, rover.Turn, mux.Vars(r))
}

func HandleReverseTurn(w http.ResponseWriter, r *http.Request) {
	invokeHaandler(w, rover.ReverseTurn, mux.Vars(r))
}

func HandleStep(w http.ResponseWriter, r *http.Request) {
	invokeHaandler(w, rover.Step, mux.Vars(r))
}

func HandleWheelStep(w http.ResponseWriter, r *http.Request) {
	invokeHaandler(w, rover.WheelStep, mux.Vars(r))
}

func HandleLightOn(w http.ResponseWriter, r *http.Request) {
	invokeHaandler(w, rover.LightOn, mux.Vars(r))
}

func HandleLightColor(w http.ResponseWriter, r *http.Request) {
	invokeHaandler(w, rover.LightColor, mux.Vars(r))
}

func HandleLightOff(w http.ResponseWriter, r *http.Request) {
	invokeHaandler(w, rover.LightOff, mux.Vars(r))
}

func HandlePlayToneFor(w http.ResponseWriter, r *http.Request) {
	invokeHaandler(w, rover.PlayToneFor, mux.Vars(r))
}

func HandlePlayTone(w http.ResponseWriter, r *http.Request) {
	invokeHaandler(w, rover.PlayTone, mux.Vars(r))
}

func HandleBuzzerOff(w http.ResponseWriter, r *http.Request) {
	invokeHaandler(w, rover.BuzzerOff, mux.Vars(r))
}

func HandleBeep(w http.ResponseWriter, r *http.Request) {
	invokeHaandler(w, rover.Beep, mux.Vars(r))
}

func HandleReadLineSensor(w http.ResponseWriter, r *http.Request) {
	invokeHaandler(w, rover.ReadLineSensor, mux.Vars(r))
}

func HandleCrossDomainReq(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handling crossdomain.xml request ...")
	fmt.Fprintln(w, "<cross-domain-policy>")
	fmt.Fprintln(w, "<allow-access-from domain=\"*\" to-ports=\"45678\"/>")
	fmt.Fprintln(w, "</cross-domain-policy>")
}

func main() {

	comPort = os.Args[1]

	fmt.Println("Expecting to find board on ", comPort)
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/crossdomain.xml", HandleCrossDomainReq)
	router.HandleFunc("/poll", HandlePoll)
	router.HandleFunc("/reset_all", HandleReset)
	router.HandleFunc("/readSonar/{id}", HandleReadSonar)
	router.HandleFunc("/turnSonar/{id}/{dir}/{angle}", HandleTurnSonar)
	router.HandleFunc("/centerSonar/{id}", HandleCenterSonar)
	router.HandleFunc("/run/{dir}", HandleRun)
	router.HandleFunc("/stop", HandleStop)
	router.HandleFunc("/turn/{id}/{dir}/{angle}", HandleTurn)
	router.HandleFunc("/turnCalibrate/{id}/{dir}/{angle}/{steps}", HandleTurnCalibrate)
	router.HandleFunc("/reverseTurn/{id}/{dir}/{angle}", HandleReverseTurn)
	router.HandleFunc("/step/{id}/{dir}/{steps}", HandleStep)
	router.HandleFunc("/wheelStep/{id}/{which}/{dir}/{steps}", HandleWheelStep)
	router.HandleFunc("/lightOn/{red}/{green}/{blue}", HandleLightOn)
	router.HandleFunc("/lightColor/{color}", HandleLightColor)
	router.HandleFunc("/lightOff", HandleLightOff)
	router.HandleFunc("/playToneFor/{id}/{freq}/{delay}", HandlePlayToneFor)
	router.HandleFunc("/playTone/{freq}", HandlePlayTone)
	router.HandleFunc("/buzzerOff", HandleBuzzerOff)
	router.HandleFunc("/beep", HandleBeep)
	router.HandleFunc("/readLineSensor/{id}", HandleReadLineSensor)

	fmt.Println("Starting server ...")
	lastCmdTime = time.Now()
	log.Fatal(http.ListenAndServe(":45678", router))
}
