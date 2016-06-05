package main

import (
	"fmt"
	"github.com/sparkybots/sparky/server/board"
	"github.com/tarm/goserial"
	"os"
	"strconv"
	"time"
)

type Rover struct {
	board      *board.Board
	sonar      Sonar
	buzzer     Buzzer
	wheels     Wheels
	lineSensor LineSensor
}

func (r *Rover) Connected() bool {
	return r.board != nil
}

func (r *Rover) Setup(comPort string, respQ chan Work) error {

	fmt.Println("Connecting to board ...")

	if p, err := serial.OpenPort(&serial.Config{Name: comPort, Baud: 9600, WriteTimeout: time.Millisecond * 300}); err == nil {
		fmt.Println("Connected successfully, initializing firmata ...")
		r.board = board.New()

		if err := r.board.Connect(p); err != nil {
			r.board = nil
			fmt.Println("Could not initialize firmata err - ", err)
			return fmt.Errorf("Could not initialize firmata err - ", err)
		} else {
			fmt.Println("Connected and initialized firmata")
			fmt.Println("firmware name:", r.board.FirmwareName)
			fmt.Println("firmata version:", r.board.ProtocolVersion)

			r.sonar = CreateSonar(r.board, respQ)
			r.buzzer = CreateBuzzer(r.board, respQ)
			r.wheels = CreateWheels(r.board, respQ)
			r.lineSensor = CreateLineSensor(r.board, respQ)

			r.Light("red")
			time.Sleep(time.Millisecond * 80)
			r.buzzer.Beep()
			time.Sleep(time.Millisecond * 500)
			r.Light("green")
			time.Sleep(time.Millisecond * 80)
			r.buzzer.Beep()
			time.Sleep(time.Millisecond * 500)
			r.buzzer.Beep()
			r.LightOff(nil)
		}

	} else {
		fmt.Println("Could not connect to board at ", comPort, err)
		return fmt.Errorf("Could not connect to board at ", comPort, err)
	}

	return nil
}

func (r *Rover) heartBeat() {
	if err := r.board.RoverHeartBeat(); err != nil {
		fmt.Println("Board is disconnected - err %s", err)
		if err := r.board.Disconnect(); err != nil {
			fmt.Println("Could not release board - err ", err)
		}
		os.Exit(1)
		return
	}
}

func (r *Rover) Reset(vars map[string]string) error {
	fmt.Println("Rover - Reset")
	return r.board.Reset()
}

func (r *Rover) ReadSonar(vars map[string]string) error {
	id := vars["id"]

	fmt.Println("Rover - ReadSonar ", id)
	return r.sonar.ReadRange(id)
}

func (r *Rover) TurnSonar(vars map[string]string) error {
	id := vars["id"]
	dir := vars["dir"]
	angle, _ := strconv.Atoi(vars["angle"])

	fmt.Println("Rover - TurnSonar ", id, dir, angle)
	return r.sonar.Turn(id, dir, angle)
}

func (r *Rover) CenterSonar(vars map[string]string) error {
	id := vars["id"]

	fmt.Println("Rover - CenterSonar ", id)
	return r.sonar.Turn(id, "left", 0)
}

func (r *Rover) Run(vars map[string]string) error {
	dir := vars["dir"]

	fmt.Println("Rover - Run ", dir)
	return r.wheels.Run(dir, 0, 0)
}

func (r *Rover) Stop(vars map[string]string) error {
	fmt.Println("Rover - Stop")
	return r.wheels.Stop()
}

var MillisPerDegreeTurn int = 6

func (r *Rover) TurnCalibrate(vars map[string]string) error {
	id := vars["id"]
	dir := vars["dir"]
	angle, _ := strconv.Atoi(vars["angle"])
	steps, _ := strconv.Atoi(vars["steps"])

	MillisPerDegreeTurn = steps

	fmt.Println("Rover - TurnCalibrate ", id, dir, angle, steps)
	return r.wheels.Turn(id, dir, angle, steps)
}

func (r *Rover) Turn(vars map[string]string) error {
	id := vars["id"]
	dir := vars["dir"]
	angle, _ := strconv.Atoi(vars["angle"])

	fmt.Println("Rover - TurnCalibrate ", id, dir, angle)
	return r.wheels.Turn(id, dir, angle, MillisPerDegreeTurn)
}

func (r *Rover) ReverseTurn(vars map[string]string) error {
	id := vars["id"]
	dir := vars["dir"]
	angle, _ := strconv.Atoi(vars["angle"])

	fmt.Println("Rover - TurnCalibrate ", id, dir, angle)
	return r.wheels.ReverseTurn(id, dir, angle, MillisPerDegreeTurn)
}

func (r *Rover) Step(vars map[string]string) error {
	id := vars["id"]
	dir := vars["dir"]
	steps, _ := strconv.Atoi(vars["steps"])

	fmt.Println("Rover - Step ", id, dir, steps)
	return r.wheels.Step(id, dir, steps)
}

func (r *Rover) WheelStep(vars map[string]string) error {
	id := vars["id"]
	which := vars["which"]
	dir := vars["dir"]
	steps, _ := strconv.Atoi(vars["steps"])

	fmt.Println("Rover - WHeelStep ", id, which, dir, steps)
	return r.wheels.WheelStep(id, which, dir, steps)
}

func (r *Rover) LightOn(vars map[string]string) error {
	red, _ := strconv.Atoi(vars["red"])
	green, _ := strconv.Atoi(vars["green"])
	blue, _ := strconv.Atoi(vars["blue"])

	fmt.Println("Rover - LightOn ", red, green, blue)
	return r.board.RoverLight(byte(red), byte(green), byte(blue))
}

func (r *Rover) Light(color string) error {
	return r.LightColor(map[string]string{"color": color})
}

func (r *Rover) LightColor(vars map[string]string) error {
	color, _ := vars["color"]

	var values []byte
	switch color {
	case "red":
		values = []byte{255, 0, 0}
	case "green":
		values = []byte{0, 255, 0}
	case "blue":
		values = []byte{0, 0, 255}
	case "yellow":
		values = []byte{255, 255, 0}
	case "cyan":
		values = []byte{0, 255, 255}
	case "magenta":
		values = []byte{255, 0, 255}
	case "white":
		values = []byte{255, 255, 255}
	}

	fmt.Println("Rover - LightColor ", color)
	return r.board.RoverLight(values[0], values[1], values[2])
}

func (r *Rover) LightOff(vars map[string]string) error {
	fmt.Println("Rover - LightOff ")
	return r.board.RoverLight(0, 0, 0)
}

func (r *Rover) PlayToneFor(vars map[string]string) error {
	id := vars["id"]
	freq, _ := strconv.Atoi(vars["freq"])
	delay, _ := strconv.Atoi(vars["delay"])
	delay = delay * 1000

	fmt.Println("Rover - PlayToneFOr ", freq, delay)
	return r.buzzer.PlayTone(id, freq, delay)
}

func (r *Rover) PlayTone(vars map[string]string) error {
	id := vars["id"]
	freq, _ := strconv.Atoi(vars["freq"])

	fmt.Println("Rover - PlayTone", freq)
	return r.buzzer.PlayTone(id, freq, 0)
}

func (r *Rover) BuzzerOff(vars map[string]string) error {
	fmt.Println("Rover - BuzzerOff")
	return r.buzzer.BuzzerOff()
}

func (r *Rover) Beep(vars map[string]string) error {
	fmt.Println("Rover - Beep")
	return r.buzzer.Beep()
}

func (r *Rover) ReadLineSensor(vars map[string]string) error {
	id := vars["id"]

	fmt.Println("Rover - ReadLineSensor ", id)
	return r.lineSensor.readLineSensors(id)
}
