// Package client provies a client for interacting with microcontrollers
// using the Firmata protocol https://github.com/firmata/protocol.
package board

import (
	"errors"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/hybridgroup/gobot"
)

// Pin Modes
const (
	Input  = 0x00
	Output = 0x01
	Analog = 0x02
	Pwm    = 0x03
	Servo  = 0x04
)

// Sysex Codes
const (
	ProtocolVersion          byte = 0xF9
	SystemReset              byte = 0xFF
	DigitalMessage           byte = 0x90
	DigitalMessageRangeStart byte = 0x90
	DigitalMessageRangeEnd   byte = 0x9F
	AnalogMessage            byte = 0xE0
	AnalogMessageRangeStart  byte = 0xE0
	AnalogMessageRangeEnd    byte = 0xEF
	ReportAnalog             byte = 0xC0
	ReportDigital            byte = 0xD0
	PinMode                  byte = 0xF4
	StartSysex               byte = 0xF0
	EndSysex                 byte = 0xF7
	CapabilityQuery          byte = 0x6B
	CapabilityResponse       byte = 0x6C
	PinStateQuery            byte = 0x6D
	PinStateResponse         byte = 0x6E
	AnalogMappingQuery       byte = 0x69
	AnalogMappingResponse    byte = 0x6A
	StringData               byte = 0x71
	I2CRequest               byte = 0x76
	I2CReply                 byte = 0x77
	I2CConfig                byte = 0x78
	FirmwareQuery            byte = 0x79
	ServoConfig              byte = 0x70

	//sysex commands for rover
	RoverSonar     byte = 0x50
	RoverMove      byte = 0x51
	RoverLED       byte = 0x52
	RoverBuzzer    byte = 0x53
	RoverHeartBeat byte = 0x54
	RoverLine      byte = 0x55
)

//I2C sub commands
const (
	I2CModeWrite          byte = 0x00
	I2CModeRead           byte = 0x01
	I2CModeContinuousRead byte = 0x02
	I2CModeStopReading    byte = 0x03
)

//Rover sub commands
const (
	SonarRead byte = 0x00
	SonarResp byte = 0x01
	SonarTurn byte = 0x02

	TurnLeft  byte = 0x00
	TurnRight byte = 0x01
	TurnResp  byte = 0x02

	MoveRun      byte = 0x00
	MoveStep     byte = 0x01
	MoveStop     byte = 0x02
	MoveTurn     byte = 0x03
	MoveTurnResp byte = 0x04
	MoveStepResp byte = 0x05

	MoveDirFwd byte = 0x00
	MoveDirRev byte = 0x01

	MoveStepBoth  byte = 0x00
	MoveStepLeft  byte = 0x01
	MoveStepRight byte = 0x02

	BuzzerPlay    byte = 0x00
	BuzzerStop    byte = 0x01
	BuzzerPlayFor byte = 0x02
	BuzzerDone    byte = 0x03
	BuzzerBeep    byte = 0x04

	LineReq  byte = 0x00
	LineResp byte = 0x01
)

// Errors
var (
	ErrConnected = errors.New("client is already connected")
)

// Board represents a client connection to a firmata board
type Board struct {
	pins             []Pin
	FirmwareName     string
	ProtocolVersion  string
	connected        bool
	connection       io.ReadWriteCloser
	analogPins       []int
	initTimeInterval time.Duration
	gobot.Eventer
}

// Pin represents a pin on the firmata board
type Pin struct {
	SupportedModes []int
	Mode           int
	Value          int
	State          int
	AnalogChannel  int
}

// I2cReply represents the response from an I2cReply message
type I2cReply struct {
	Address  int
	Register int
	Data     []byte
}

// New returns a new Board
func New() *Board {
	c := &Board{
		ProtocolVersion: "",
		FirmwareName:    "",
		connection:      nil,
		pins:            []Pin{},
		analogPins:      []int{},
		connected:       false,
		Eventer:         gobot.NewEventer(),
	}

	for _, s := range []string{
		"FirmwareQuery",
		"CapabilityQuery",
		"AnalogMappingQuery",
		"ProtocolVersion",
		"I2cReply",
		"StringData",
		"SonarResponse",
		"SonarTurnDone",
		"BuzzerDone",
		"RoverTurnDone",
		"RoverStepDone",
		"RoverLineResponse",
		"Error",
	} {
		c.AddEvent(s)
	}

	return c
}

// Disconnect disconnects the Board
func (b *Board) Disconnect() (err error) {
	b.connected = false
	return b.connection.Close()
}

// Connected returns the current connection state of the Board
func (b *Board) Connected() bool {
	return b.connected
}

// Pins returns all available pins
func (b *Board) Pins() []Pin {
	return b.pins
}

// Connect connects to the Board given conn. It first resets the firmata board
// then continuously polls the firmata board for new information when it's
// available.
func (b *Board) Connect(conn io.ReadWriteCloser) (err error) {
	if b.connected {
		return ErrConnected
	}

	b.connection = conn
	b.Reset()

	initFunc := b.ProtocolVersionQuery

	gobot.Once(b.Event("ProtocolVersion"), func(data interface{}) {
		//initFunc = b.FirmwareQuery
		initFunc = func() error { return nil }
		b.connected = true
	})

	gobot.Once(b.Event("FirmwareQuery"), func(data interface{}) {
		initFunc = b.CapabilitiesQuery
	})

	gobot.Once(b.Event("CapabilityQuery"), func(data interface{}) {
		initFunc = b.AnalogMappingQuery
	})

	gobot.Once(b.Event("AnalogMappingQuery"), func(data interface{}) {
		initFunc = func() error { return nil }
		b.ReportDigital(0, 1)
		b.ReportDigital(1, 1)
		b.connected = true
	})

	for {
		if err := initFunc(); err != nil {
			return err
		}
		if err := b.process(); err != nil {
			return err
		}
		if b.connected {
			go func() {
				for {
					if !b.connected {
						break
					}

					if err := b.process(); err != nil {
						gobot.Publish(b.Event("Error"), err)
					}
				}
			}()
			break
		}
	}
	return
}

// Reset sends the SystemReset sysex code.
func (b *Board) Reset() error {
	return b.write([]byte{SystemReset})
}

// SetPinMode sets the pin to mode.
func (b *Board) SetPinMode(pin int, mode int) error {
	b.pins[byte(pin)].Mode = mode
	return b.write([]byte{PinMode, byte(pin), byte(mode)})
}

// DigitalWrite writes value to pin.
func (b *Board) DigitalWrite(pin int, value int) error {
	port := byte(math.Floor(float64(pin) / 8))
	portValue := byte(0)

	b.pins[pin].Value = value

	for i := byte(0); i < 8; i++ {
		if b.pins[8*port+i].Value != 0 {
			portValue = portValue | (1 << i)
		}
	}
	return b.write([]byte{DigitalMessage | port, portValue & 0x7F, (portValue >> 7) & 0x7F})
}

// ServoConfig sets the min and max pulse width for servo PWM range
func (b *Board) ServoConfig(pin int, max int, min int) error {
	ret := []byte{
		ServoConfig,
		byte(pin),
		byte(max & 0x7F),
		byte((max >> 7) & 0x7F),
		byte(min & 0x7F),
		byte((min >> 7) & 0x7F),
	}
	return b.writeSysex(ret)
}

// AnalogWrite writes value to pin.
func (b *Board) AnalogWrite(pin int, value int) error {
	b.pins[pin].Value = value
	return b.write([]byte{AnalogMessage | byte(pin), byte(value & 0x7F), byte((value >> 7) & 0x7F)})
}

// FirmwareQuery sends the FirmwareQuery sysex code.
func (b *Board) FirmwareQuery() error {
	return b.writeSysex([]byte{FirmwareQuery})
}

// PinStateQuery sends a PinStateQuery for pin.
func (b *Board) PinStateQuery(pin int) error {
	return b.writeSysex([]byte{PinStateQuery, byte(pin)})
}

// ProtocolVersionQuery sends the ProtocolVersion sysex code.
func (b *Board) ProtocolVersionQuery() error {
	return b.write([]byte{ProtocolVersion})
}

// CapabilitiesQuery sends the CapabilityQuery sysex code.
func (b *Board) CapabilitiesQuery() error {
	return b.writeSysex([]byte{CapabilityQuery})
}

// AnalogMappingQuery sends the AnalogMappingQuery sysex code.
func (b *Board) AnalogMappingQuery() error {
	return b.writeSysex([]byte{AnalogMappingQuery})
}

// ReportDigital enables or disables digital reporting for pin, a non zero
// state enables reporting
func (b *Board) ReportDigital(pin int, state int) error {
	return b.togglePinReporting(pin, state, ReportDigital)
}

// ReportAnalog enables or disables analog reporting for pin, a non zero
// state enables reporting
func (b *Board) ReportAnalog(pin int, state int) error {
	return b.togglePinReporting(pin, state, ReportAnalog)
}

// I2cRead reads numBytes from address once.
func (b *Board) I2cRead(address int, numBytes int) error {
	return b.writeSysex([]byte{I2CRequest, byte(address), (I2CModeRead << 3),
		byte(numBytes) & 0x7F, (byte(numBytes) >> 7) & 0x7F})
}

// I2cWrite writes data to address.
func (b *Board) I2cWrite(address int, data []byte) error {
	ret := []byte{I2CRequest, byte(address), (I2CModeWrite << 3)}
	for _, val := range data {
		ret = append(ret, byte(val&0x7F))
		ret = append(ret, byte((val>>7)&0x7F))
	}
	return b.writeSysex(ret)
}

// I2cConfig configures the delay in which a register can be read from after it
// has been written to.
func (b *Board) I2cConfig(delay int) error {
	return b.writeSysex([]byte{I2CConfig, byte(delay & 0xFF), byte((delay >> 8) & 0xFF)})
}

func (b *Board) RoverSonarRead() error {
	return b.writeSysex([]byte{RoverSonar, SonarRead})
}

func (b *Board) RoverSonarTurn(dir byte, angle int) error {
	return b.writeSysex([]byte{RoverSonar, SonarTurn, dir, byte(angle & 0x7F), byte((angle >> 7) & 0x7F)})
}

func (b *Board) RoverRun(dir byte, leftSpeed byte, rightSpeed byte) error {
	if leftSpeed == 0 && rightSpeed == 0 {
		return b.writeSysex([]byte{RoverMove, MoveRun, dir})
	} else {
		return b.writeSysex([]byte{RoverMove, MoveRun, dir, byte(leftSpeed & 0x7F), byte((leftSpeed >> 7) & 0x7F), byte(rightSpeed & 0x7F), byte((rightSpeed >> 7) & 0x7F)})
	}
}

func (b *Board) RoverStop() error {
	return b.writeSysex([]byte{RoverMove, MoveStop})
}

func (b *Board) RoverTurn(side byte, dir byte, angle byte, steps int) error {
	return b.writeSysex([]byte{RoverMove, MoveTurn, side, dir, angle, byte(steps & 0x7F), byte((steps >> 7) & 0x7F)})
}

func (b *Board) RoverStep(dir byte, steps int) error {
	return b.writeSysex([]byte{RoverMove, MoveStep, MoveStepBoth, dir, byte(steps & 0x7F), byte((steps >> 7) & 0x7F)})
}

func (b *Board) RoverWheelStep(which byte, dir byte, steps int) error {
	return b.writeSysex([]byte{RoverMove, MoveStep, which, dir, byte(steps & 0x7F), byte((steps >> 7) & 0x7F)})
}

func (b *Board) RoverLight(red byte, green byte, blue byte) error {
	data := []byte{
		RoverLED,
		byte(red & 0x7F),
		byte((red >> 7) & 0x7F),
		byte(green & 0x7F),
		byte((green >> 7) & 0x7F),
		byte(blue & 0x7F),
		byte((blue >> 7) & 0x7F),
	}
	return b.writeSysex(data)
}

func (b *Board) RoverPlayTone(freq byte, delay int) error {
	switch delay {
	case 0:
		return b.writeSysex([]byte{RoverBuzzer, BuzzerPlay, byte(freq & 0x7F), byte((freq >> 7) & 0x7F)})
	default:
		return b.writeSysex([]byte{RoverBuzzer, BuzzerPlayFor, byte(freq & 0x7F), byte((freq >> 7) & 0x7F), byte(delay & 0x7F), byte((delay >> 7) & 0x7F)})
	}
}

func (b *Board) RoverBuzzerOff() error {
	return b.writeSysex([]byte{RoverBuzzer, BuzzerStop})
}

func (b *Board) RoverBeep() error {
	return b.writeSysex([]byte{RoverBuzzer, BuzzerBeep})
}

func (b *Board) RoverHeartBeat() error {
	return b.writeSysex([]byte{RoverHeartBeat, 0x0, 0x1, 0x02, 0x3, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x20})
}

func (b *Board) RoverReadLineSensors() error {
	return b.writeSysex([]byte{RoverLine, LineReq})
}

func (b *Board) togglePinReporting(pin int, state int, mode byte) error {
	if state != 0 {
		state = 1
	} else {
		state = 0
	}

	if err := b.write([]byte{byte(mode) | byte(pin), byte(state)}); err != nil {
		return err
	}

	return nil

}

func (b *Board) writeSysex(data []byte) (err error) {
	return b.write(append([]byte{StartSysex}, append(data, EndSysex)...))
}

func (b *Board) write(data []byte) (err error) {
	n, err := b.connection.Write(data[:])
	if n < len(data) {
		err = fmt.Errorf("Could not write requested bytes err: %s", err)
	}
	return
}

func (b *Board) read(length int) (buf []byte, err error) {
	i := 0
	for length > 0 {
		tmp := make([]byte, length)
		if i, err = b.connection.Read(tmp); err != nil {
			if err.Error() != "EOF" {
				return
			}
			<-time.After(5 * time.Millisecond)
		}
		if i > 0 {
			buf = append(buf, tmp[0:i]...)
			length = length - i
		}
	}
	return
}

func (b *Board) process() (err error) {
	buf, err := b.read(3)
	if err != nil {
		return err
	}
	messageType := buf[0]
	fmt.Printf("Received firmata msg: %X\n", messageType)
	switch {
	case ProtocolVersion == messageType:
		fmt.Println("ProtocolVersion")
		b.ProtocolVersion = fmt.Sprintf("%v.%v", buf[1], buf[2])

		gobot.Publish(b.Event("ProtocolVersion"), b.ProtocolVersion)
	case AnalogMessageRangeStart <= messageType &&
		AnalogMessageRangeEnd >= messageType:
		fmt.Println("AnalogMessage")

		value := uint(buf[1]) | uint(buf[2])<<7
		pin := int((messageType & 0x0F))

		if len(b.analogPins) > pin {
			if len(b.pins) > b.analogPins[pin] {
				b.pins[b.analogPins[pin]].Value = int(value)
				gobot.Publish(b.Event(fmt.Sprintf("AnalogRead%v", pin)), b.pins[b.analogPins[pin]].Value)
			}
		}
	case DigitalMessageRangeStart <= messageType &&
		DigitalMessageRangeEnd >= messageType:
		fmt.Println("DigitalMessage")

		port := messageType & 0x0F
		portValue := buf[1] | (buf[2] << 7)

		for i := 0; i < 8; i++ {
			pinNumber := int((8*byte(port) + byte(i)))
			if len(b.pins) > pinNumber {
				if b.pins[pinNumber].Mode == Input {
					b.pins[pinNumber].Value = int((portValue >> (byte(i) & 0x07)) & 0x01)
					gobot.Publish(b.Event(fmt.Sprintf("DigitalRead%v", pinNumber)), b.pins[pinNumber].Value)
				}
			}
		}
	case StartSysex == messageType:

		currentBuffer := buf
		for {
			buf, err = b.read(1)
			if err != nil {
				return err
			}
			currentBuffer = append(currentBuffer, buf[0])
			if buf[0] == EndSysex {
				break
			}
		}
		command := currentBuffer[1]
		fmt.Printf("Received firmata SYSEX msg: ")
		for _, val := range currentBuffer {
			fmt.Printf(" %X", val)
		}
		fmt.Println("")

		switch command {
		case CapabilityResponse:
			fmt.Println("CapabilityResponse")
			b.pins = []Pin{}
			supportedModes := 0
			n := 0

			for _, val := range currentBuffer[2:(len(currentBuffer) - 5)] {
				if val == 127 {
					modes := []int{}
					for _, mode := range []int{Input, Output, Analog, Pwm, Servo} {
						if (supportedModes & (1 << byte(mode))) != 0 {
							modes = append(modes, mode)
						}
					}

					b.pins = append(b.pins, Pin{SupportedModes: modes, Mode: Output})
					b.AddEvent(fmt.Sprintf("DigitalRead%v", len(b.pins)-1))
					b.AddEvent(fmt.Sprintf("PinState%v", len(b.pins)-1))
					supportedModes = 0
					n = 0
					continue
				}

				if n == 0 {
					supportedModes = supportedModes | (1 << val)
				}
				n ^= 1
			}
			gobot.Publish(b.Event("CapabilityQuery"), nil)
		case AnalogMappingResponse:
			fmt.Println("AnalogMappingResponse")
			pinIndex := 0
			b.analogPins = []int{}

			for _, val := range currentBuffer[2 : len(b.pins)-1] {

				b.pins[pinIndex].AnalogChannel = int(val)

				if val != 127 {
					b.analogPins = append(b.analogPins, pinIndex)
				}
				b.AddEvent(fmt.Sprintf("AnalogRead%v", pinIndex))
				pinIndex++
			}
			gobot.Publish(b.Event("AnalogMappingQuery"), nil)
		case PinStateResponse:
			fmt.Println("PrintStateResponse")
			pin := currentBuffer[2]
			b.pins[pin].Mode = int(currentBuffer[3])
			b.pins[pin].State = int(currentBuffer[4])

			if len(currentBuffer) > 6 {
				b.pins[pin].State = int(uint(b.pins[pin].State) | uint(currentBuffer[5])<<7)
			}
			if len(currentBuffer) > 7 {
				b.pins[pin].State = int(uint(b.pins[pin].State) | uint(currentBuffer[6])<<14)
			}

			gobot.Publish(b.Event(fmt.Sprintf("PinState%v", pin)), b.pins[pin])
		case I2CReply:
			fmt.Println("I2CReplay")
			reply := I2cReply{
				Address:  int(byte(currentBuffer[2]) | byte(currentBuffer[3])<<7),
				Register: int(byte(currentBuffer[4]) | byte(currentBuffer[5])<<7),
				Data:     []byte{byte(currentBuffer[6]) | byte(currentBuffer[7])<<7},
			}
			for i := 8; i < len(currentBuffer); i = i + 2 {
				if currentBuffer[i] == byte(0xF7) {
					break
				}
				if i+2 > len(currentBuffer) {
					break
				}
				reply.Data = append(reply.Data,
					byte(currentBuffer[i])|byte(currentBuffer[i+1])<<7,
				)
			}
			gobot.Publish(b.Event("I2cReply"), reply)
		case FirmwareQuery:
			fmt.Println("FirmwareQuery")
			name := []byte{}
			for _, val := range currentBuffer[4:(len(currentBuffer) - 1)] {
				if val != 0 {
					name = append(name, val)
				}
			}
			b.FirmwareName = string(name[:])
			gobot.Publish(b.Event("FirmwareQuery"), b.FirmwareName)
		case StringData:
			fmt.Println("StringData")
			str := currentBuffer[2:len(currentBuffer)]
			gobot.Publish(b.Event("StringData"), string(str[:len(str)-1]))
		case RoverSonar:
			fmt.Println("Received sonar response")
			oper := currentBuffer[2]
			switch oper {
			case SonarResp:
				distance := (currentBuffer[4] << 7) | currentBuffer[3]
				gobot.Publish(b.Event("SonarResponse"), distance)
			case SonarTurn:
				if currentBuffer[3] == TurnResp {
					gobot.Publish(b.Event("SonarTurnDone"), nil)
				}
			}
		case RoverBuzzer:
			fmt.Println("Received buzzer response")
			oper := currentBuffer[2]
			if oper == BuzzerDone {
				gobot.Publish(b.Event("BuzzerDone"), nil)
			}
		case RoverMove:
			fmt.Println("Received turn/step response")
			oper := currentBuffer[2]
			switch oper {
			case MoveTurnResp:
				gobot.Publish(b.Event("RoverTurnDone"), nil)
			case MoveStepResp:
				gobot.Publish(b.Event("RoverStepDone"), nil)
			}
		case RoverLine:
			fmt.Println("Recived line response")
			oper := currentBuffer[2]
			if oper == LineResp {
				left := currentBuffer[3]
				right := currentBuffer[4]
				value := (right << 1) | left
				gobot.Publish(b.Event("RoverLineResponse"), value)
			}
		}
	}
	return
}
