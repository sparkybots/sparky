
/*
 SparkyBot firmware

 Communicates to controlling PC using firmata protoccol
*/

#include <Wire.h>
#include <Firmata.h>
#include <SoftwareSerial.h>
#include <NewPing.h>
#include <SoftwareServo.h>
#include <Timer.h>

//User defined sysex commands
#define ROVER_SONAR     0x50
#define ROVER_MOVE      0x51
#define ROVER_LED       0x52
#define ROVER_BUZZER    0x53
#define ROVER_HEARTBEAT 0x54
#define ROVER_LINE      0x55

// sonar sub commands
#define SONAR_READ   0x00
#define SONAR_RESP   0x01
#define SONAR_TURN   0x02

// sonar turn sub commands
#define TURN_LEFT    0x00
#define TURN_RIGHT   0x01
#define TURN_RESP    0x02

// move sub commands
#define MOVE_RUN         0x00
#define MOVE_STEP        0x01
#define MOVE_STOP        0x02
#define MOVE_TURN        0x03
#define MOVE_TURN_RESP   0x04
#define MOVE_STEP_RESP   0x05

#define MOVE_DIR_FWD 0x00
#define MOVE_DIR_REV 0x01

#define MOVE_STEP_BOTH  0x00
#define MOVE_STEP_LEFT  0x01
#define MOVE_STEP_RIGHT 0x02

#define BUZZER_PLAY    0x00
#define BUZZER_OFF     0x01
#define BUZZER_PLAYFOR 0x02
#define BUZZER_DONE    0x03
#define BUZZER_BEEP    0x04

#define LINE_REQ       0x00
#define LINE_RESP      0x01

#define BLUSERIAL_TX 0
#define BLUSERIAL_RX 1

#define SERIAL_RX 0
#define SERIAL_TX 1

#define SONAR_PIN_TRIG  A0
#define SONAR_PIN_ECHO  A1
#define SONAR_MAX_DISTANCE 200

#define HEAD_SERVO_PIN 3
#define HEAD_CENTER    95

#define WHEEL_LEFT_DIR   4
#define WHEEL_RIGHT_DIR  7

#define WHEEL_LEFT_SPEED  5
#define WHEEL_RIGHT_SPEED 6

#define LIGHT_RED   9
#define LIGHT_GREEN 10
#define LIGHT_BLUE  11

#define LINE_LEFT  A3
#define LINE_RIGHT A2

#define BUZZER_PIN  8

SoftwareSerial BluSerial(BLUSERIAL_RX, BLUSERIAL_TX); // RX, TX
NewPing sonar(SONAR_PIN_TRIG, SONAR_PIN_ECHO, SONAR_MAX_DISTANCE);
SoftwareServo head;
Timer softwareTimer;

/*==============================================================================
 * SYSEX-BASED commands
 *============================================================================*/

void reportSonarRange() 
{
  int uS, uS1, uS2, uS3, range;
  byte resp[2];
  uS1 = sonar.ping();
  delay(30);
  uS2 = sonar.ping();
  delay(30);
  uS3 = sonar.ping();

  uS = (uS1+uS2+uS3)/3.0;
  range = uS/ US_ROUNDTRIP_CM;

  if (range == 0) {
      range = SONAR_MAX_DISTANCE;
  }
  resp[0]  = range & 0x7F;
  resp[1]  = (range >> 7) & 0x7F;
  
  Firmata.write(START_SYSEX);
  Firmata.write(ROVER_SONAR);
  Firmata.write(SONAR_RESP);
  Firmata.write(resp[0]);
  Firmata.write(resp[1]);
  Firmata.write(END_SYSEX);
}

void headCenterDone() {
  head.detach();
}

void centerHead() {
  head.attach(HEAD_SERVO_PIN);
  head.write(HEAD_CENTER);
  softwareTimer.after(800, headCenterDone );
}

void sonarTurnDone() {
  head.detach();
  Firmata.write(START_SYSEX);
  Firmata.write(ROVER_SONAR);
  Firmata.write(SONAR_TURN);
  Firmata.write(TURN_RESP);
  Firmata.write(END_SYSEX);  
}

void turnSonar(int direction, int angle) {
  angle = angle % 91;
  head.attach(HEAD_SERVO_PIN);
  switch (direction) {
  case TURN_LEFT:
      head.write(HEAD_CENTER+angle);
      break;
  case TURN_RIGHT:
      head.write(HEAD_CENTER-angle);
      break;      
  }  
  softwareTimer.after(800, sonarTurnDone);
}

int PWM_LEFT = 200;
int PWM_RIGHT = 200;

void roverRun(int dir, int left, int right) {
  if (left == 0) { left = PWM_LEFT; }
  if (right == 0) { right = PWM_RIGHT; }
  
  switch (dir) {
  case MOVE_DIR_FWD:
      digitalWrite(WHEEL_LEFT_DIR, HIGH);
      digitalWrite(WHEEL_RIGHT_DIR, HIGH);

      analogWrite(WHEEL_RIGHT_SPEED, 255 - PWM_RIGHT);
      analogWrite(WHEEL_LEFT_SPEED, 255 - PWM_LEFT);
      break;
  case MOVE_DIR_REV:
      digitalWrite(WHEEL_LEFT_DIR, LOW);
      digitalWrite(WHEEL_RIGHT_DIR, LOW);
      
      analogWrite(WHEEL_RIGHT_SPEED, PWM_RIGHT);
      analogWrite(WHEEL_LEFT_SPEED, PWM_LEFT);
      break;
  }
}

void roverStop() {
  digitalWrite(WHEEL_RIGHT_SPEED, LOW);
  digitalWrite(WHEEL_LEFT_SPEED, LOW);   
  digitalWrite(WHEEL_LEFT_DIR, LOW);
  digitalWrite(WHEEL_RIGHT_DIR, LOW);  
}

void roverTurnDone() {
 roverStop(); 
 Firmata.write(START_SYSEX);
 Firmata.write(ROVER_MOVE); 
 Firmata.write(MOVE_TURN_RESP);
 Firmata.write(END_SYSEX);   
}

void roverTurn(byte side, byte dir, byte angle, int steps) {
  int duration = angle * steps;

  switch (dir) {
    case MOVE_DIR_FWD:
        switch(side) {
          case TURN_LEFT:
            digitalWrite(WHEEL_LEFT_DIR, LOW);
            digitalWrite(WHEEL_RIGHT_DIR, HIGH);

            analogWrite(WHEEL_RIGHT_SPEED, 255 - PWM_RIGHT);
            digitalWrite(WHEEL_LEFT_SPEED, LOW); 
            break;
          case TURN_RIGHT:
            digitalWrite(WHEEL_LEFT_DIR, HIGH);
            digitalWrite(WHEEL_RIGHT_DIR, LOW);
     
            digitalWrite(WHEEL_RIGHT_SPEED, LOW);
            analogWrite(WHEEL_LEFT_SPEED, 255 - PWM_LEFT);         
            break;
        }
        break;
    case MOVE_DIR_REV:
        switch(side) {
          case TURN_LEFT:
            digitalWrite(WHEEL_LEFT_DIR, LOW);
            digitalWrite(WHEEL_RIGHT_DIR, LOW);

            analogWrite(WHEEL_RIGHT_SPEED, PWM_RIGHT);
            digitalWrite(WHEEL_LEFT_SPEED, LOW); 
            break;
          case TURN_RIGHT:
            digitalWrite(WHEEL_LEFT_DIR, LOW);
            digitalWrite(WHEEL_RIGHT_DIR, LOW);
     
            digitalWrite(WHEEL_RIGHT_SPEED, LOW);
            analogWrite(WHEEL_LEFT_SPEED, PWM_LEFT);         
            break;
        }
        break;
  }

 softwareTimer.after(duration, roverTurnDone);
}

void stepDone() {
 roverStop(); 
 Firmata.write(START_SYSEX);
 Firmata.write(ROVER_MOVE); 
 Firmata.write(MOVE_STEP_RESP);
 Firmata.write(END_SYSEX);     
}

void roverStep(byte dir, byte which, int steps) {
 int duration = 60*steps;
 switch (which) {
   case MOVE_STEP_BOTH:
     roverRun(dir, 0, 0);
     break;
   case MOVE_STEP_LEFT:
     digitalWrite(WHEEL_RIGHT_DIR, LOW);
     analogWrite(WHEEL_RIGHT_SPEED, LOW);

     if (dir == MOVE_DIR_FWD) {
       digitalWrite(WHEEL_LEFT_DIR, HIGH); 
       analogWrite(WHEEL_LEFT_SPEED, 255 - PWM_LEFT);    
     } else {
       digitalWrite(WHEEL_LEFT_DIR, LOW); 
       analogWrite(WHEEL_LEFT_SPEED, PWM_LEFT);    
     }
     break;
   case MOVE_STEP_RIGHT:
     digitalWrite(WHEEL_LEFT_DIR, LOW);
     analogWrite(WHEEL_LEFT_SPEED, LOW);

     if (dir == MOVE_DIR_FWD) {
       digitalWrite(WHEEL_RIGHT_DIR, HIGH);     
       analogWrite(WHEEL_RIGHT_SPEED, 255 - PWM_RIGHT);
     } else {
       digitalWrite(WHEEL_RIGHT_DIR, LOW);     
       analogWrite(WHEEL_RIGHT_SPEED, PWM_RIGHT);      
     }          
     break;
 } 
 softwareTimer.after(duration, stepDone);
}

void roverLight(byte red, byte green, byte blue) {
  analogWrite(LIGHT_RED, red);
  analogWrite(LIGHT_GREEN, green);
  analogWrite(LIGHT_BLUE, blue);
}

void buzzerDone() {
  buzzerOff();
  Firmata.write(START_SYSEX);
  Firmata.write(ROVER_BUZZER);
  Firmata.write(BUZZER_DONE);  
  Firmata.write(END_SYSEX);
}

void playToneFor(byte freq, int delayms) {
  pinMode(BUZZER_PIN, OUTPUT);
  tone(BUZZER_PIN, freq);
  softwareTimer.after(delayms, buzzerDone);
}

void playTone(byte freq) {
  pinMode(BUZZER_PIN, OUTPUT);
  tone(BUZZER_PIN, freq);
}

void buzzerOff() {  
 noTone(BUZZER_PIN); 
 pinMode(BUZZER_PIN, INPUT);
}

void buzzerBeep() {
 pinMode(BUZZER_PIN, OUTPUT);
 tone(BUZZER_PIN, 30);  
 softwareTimer.after(100, buzzerOff);
}

void roverReportLineReadings() {
  byte lineRight, lineLeft;

  lineLeft = digitalRead(LINE_LEFT);
  lineRight = digitalRead(LINE_RIGHT);

  Firmata.write(START_SYSEX);
  Firmata.write(ROVER_LINE);
  Firmata.write(LINE_RESP);  
  Firmata.write(lineLeft);
  Firmata.write(lineRight);
  Firmata.write(END_SYSEX);
}

void sysexCallback(byte command, byte argc, byte *argv)
{
  byte mode;
  byte stopTX;
  byte slaveAddress;
  byte data;
  int slaveRegister;
  unsigned int delayTime;

  switch (command) {
    case CAPABILITY_QUERY:
      Firmata.write(START_SYSEX);
      Firmata.write(CAPABILITY_RESPONSE);
      for (byte pin = 0; pin < TOTAL_PINS; pin++) {
        if (IS_PIN_DIGITAL(pin)) {
          Firmata.write((byte)INPUT);
          Firmata.write(1);
          Firmata.write((byte)PIN_MODE_PULLUP);
          Firmata.write(1);
          Firmata.write((byte)OUTPUT);
          Firmata.write(1);
        }
        if (IS_PIN_ANALOG(pin)) {
          Firmata.write(PIN_MODE_ANALOG);
          Firmata.write(10); // 10 = 10-bit resolution
        }
        if (IS_PIN_PWM(pin)) {
          Firmata.write(PIN_MODE_PWM);
          Firmata.write(8); // 8 = 8-bit resolution
        }
        if (IS_PIN_DIGITAL(pin)) {
          Firmata.write(PIN_MODE_SERVO);
          Firmata.write(14);
        }
        if (IS_PIN_I2C(pin)) {
          Firmata.write(PIN_MODE_I2C);
          Firmata.write(1);  // TODO: could assign a number to map to SCL or SDA
        }
#ifdef FIRMATA_SERIAL_FEATURE
        serialFeature.handleCapability(pin);
#endif
        Firmata.write(127);
      }
      Firmata.write(END_SYSEX);
      break;
    case PIN_STATE_QUERY:
      if (argc > 0) {
        byte pin = argv[0];
        Firmata.write(START_SYSEX);
        Firmata.write(PIN_STATE_RESPONSE);
        Firmata.write(pin);
        if (pin < TOTAL_PINS) {
          Firmata.write(Firmata.getPinMode(pin));
          Firmata.write((byte)Firmata.getPinState(pin) & 0x7F);
          if (Firmata.getPinState(pin) & 0xFF80) Firmata.write((byte)(Firmata.getPinState(pin) >> 7) & 0x7F);
          if (Firmata.getPinState(pin) & 0xC000) Firmata.write((byte)(Firmata.getPinState(pin) >> 14) & 0x7F);
        }
        Firmata.write(END_SYSEX);
      }
      break;
    case ANALOG_MAPPING_QUERY:
      Firmata.write(START_SYSEX);
      Firmata.write(ANALOG_MAPPING_RESPONSE);
      for (byte pin = 0; pin < TOTAL_PINS; pin++) {
        Firmata.write(IS_PIN_ANALOG(pin) ? PIN_TO_ANALOG(pin) : 127);
      }
      Firmata.write(END_SYSEX);
      break;
      
    case ROVER_SONAR:
       byte angle;
       switch (argv[0]) {
       case SONAR_READ:         
          reportSonarRange();     
          break;
       case SONAR_TURN:
          angle = argv[2] | (argv[3] << 7);
          turnSonar(argv[1], angle);
          break;
       default:   
          break;     
       }
       break;
    case ROVER_MOVE:
       int dir, side, which;
       int left, right, steps;
       switch (argv[0]) {
       case MOVE_RUN:
           dir = argv[1];
           if (argc > 2) {
               left = argv[2] | (argv[3] << 7);
               right = argv[4] | (argv[5] << 7);
           } else {
               left = 0;
               right = 0;
           }
           roverRun(dir, left, right);
           break;
       case MOVE_STOP:
           roverStop();
           break;
       case MOVE_TURN:
           side = argv[1];
           dir = argv[2];
           angle = argv[3];
           steps = argv[4] | (argv[5] << 7);
           roverTurn(side, dir, angle, steps);
           break;
       case MOVE_STEP:
          which = argv[1];
          dir = argv[2];
          steps = argv[3] | (argv[4] << 7);
          roverStep(dir, which, steps);
          break;
       } 
       break;
    case ROVER_LED:
       byte red, green, blue;
       red   = argv[0] | (argv[1] << 7);
       green = argv[2] | (argv[3] << 7);
       blue  = argv[4] | (argv[5] << 7);
       roverLight(red, green, blue);
       break;
    case ROVER_BUZZER:
       byte freq;
       int delayms;
       switch (argv[0]) {
       case BUZZER_PLAY:
           freq = argv[1] | (argv[2] << 7);
           playTone(freq);
           break;
       case BUZZER_OFF:
           buzzerOff();
           break;
       case BUZZER_PLAYFOR:
           freq = argv[1] | (argv[2] << 7);
           delayms = argv[3] | ( argv[4] << 7);
           playToneFor(freq, delayms);
           break;
       case BUZZER_BEEP:
           buzzerBeep();
           break;
       }
       break;
   case ROVER_HEARTBEAT:
       break;
   case ROVER_LINE:
      roverReportLineReadings();
      break;
  }
}

/*==============================================================================
 * SETUP()
 *============================================================================*/

void systemResetCallback()
{

  roverStop();
  centerHead();
  roverLight(0,0,0);
  buzzerOff();

  PWM_LEFT = 200;
  PWM_RIGHT = 200;  
}

void setup()
{
  Firmata.setFirmwareVersion(FIRMATA_FIRMWARE_MAJOR_VERSION, FIRMATA_FIRMWARE_MINOR_VERSION);
  
  Firmata.attach(START_SYSEX, sysexCallback);
  Firmata.attach(SYSTEM_RESET, systemResetCallback);

  pinMode(WHEEL_RIGHT_SPEED, OUTPUT);
  pinMode(WHEEL_LEFT_SPEED, OUTPUT);
  pinMode(WHEEL_LEFT_DIR, OUTPUT);
  pinMode(WHEEL_RIGHT_DIR, OUTPUT);
  
  digitalWrite(WHEEL_RIGHT_SPEED, LOW);
  digitalWrite(WHEEL_LEFT_SPEED, LOW);       
  digitalWrite(WHEEL_RIGHT_DIR, LOW);
  digitalWrite(WHEEL_LEFT_DIR, LOW);
  
  pinMode(LIGHT_RED, OUTPUT);
  pinMode(LIGHT_GREEN, OUTPUT);
  pinMode(LIGHT_BLUE, OUTPUT);

  pinMode(LINE_LEFT, INPUT);
  pinMode(LINE_RIGHT, INPUT);
  
  // to use a port other than Serial, such as Serial1 on an Arduino Leonardo or Mega,
  // Call begin(baud) on the alternate serial port and pass it to Firmata to begin like this:
  // Serial1.begin(57600);
  // Firmata.begin(Serial1);
  // However do not do this if you are using SERIAL_MESSAGE
  
  //Firmata.begin(9600);
  //while (!Serial) {
  //   ; // wait for serial port to connect. Needed for ATmega32u4-based boards and Arduino 101
  //  }  

  //Serial.begin(9600);
  BluSerial.begin(9600);
  Firmata.begin(BluSerial);
  
  systemResetCallback();  // reset to default config

  PWM_LEFT = 200;
  PWM_RIGHT = 200;

}

/*==============================================================================
 * LOOP()
 *============================================================================*/
void loop()
{
  while (Firmata.available()) {
    Firmata.processInput();
  }

  softwareTimer.update();
  SoftwareServo::refresh();
}

