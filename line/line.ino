#define DEBUG
#include <Adafruit_MotorShield.h>
#include <EEPROM.h>

#define VARIANT "v2"
// Instance is initially set to this, if nothing exists at EEPROM_INSTANCE_ADDR.
#define INSTANCE "4"

// === EEPROM Layout
// 00-10  variant string
// 10-31  instance name (null-terminated)
#define EEPROM_VERSION_ADDR 0x0
#define EEPROM_INSTANCE_ADDR 0x10
#include "ina240.h"

typedef struct Line {
  char id;
  Adafruit_DCMotor *motor;
  bool direction;
  unsigned long stop_ms;
  bool stop_brake;
  int pwm;
} Line;

static char instance[0x21] = INSTANCE;
Adafruit_MotorShield shield0 = Adafruit_MotorShield();

static Line lineA = {
    .id = 'A',
    .motor = shield0.getMotor(1),
    .direction = true,
    .stop_ms = 0,
};
static Line lineB = {
    .id = 'B',
    .motor = shield0.getMotor(2),
    .direction = true,
    .stop_ms = 0,
};
static Line lineC = {
    .id = 'C',
    .motor = shield0.getMotor(3),
    .direction = true,
    .stop_ms = 0,
};
static Line lineD = {
    .id = 'D',
    .motor = shield0.getMotor(4),
    .direction = true,
    .stop_ms = 0,
};

void Line_setDirection(Line *line, bool direction) {
  line->direction = direction;
}

void Line_setPwm(Line *line, int value, bool brake) {
  line->motor->setSpeed(value);
  if (!value) {
    line->motor->run(RELEASE);
    return;
  }
  line->motor->run(brake ? BRAKE : (line->direction ? FORWARD : BACKWARD));
  line->pwm = value;
}

void Line_update(Line *line) {
  unsigned long now = millis();
  if (0 != line->stop_ms && now > line->stop_ms) {
    Line_setPwm(line, 0, line->stop_brake);
    // TODO: send confirmation
    Serial.print(" DSL");
    Serial.print(line->id);
    Serial.print("T");
    Serial.println(now);
    line->stop_ms = 0;
  }
}

void setup() {
  Serial.begin(9600);
  Serial.println(" Sstart");
  if (EEPROM.read(EEPROM_INSTANCE_ADDR) == 0) {
    Serial.println(" SInitialising EEPROM with instance name...");
    static char *newInstance = instance;
    for (size_t i = 0; i < 0x20; i++) {
      EEPROM.update(EEPROM_INSTANCE_ADDR + i,
                    i < strlen(newInstance) ? newInstance[i] : 0);
    }
    Serial.println(" SInitialised EEPROM with instance name.");
  } else {
    for (size_t i = 0; i < 0x20; i++) {
      instance[i] = EEPROM.read(EEPROM_INSTANCE_ADDR + i);
    }
  }
  // === Motor Shield
  while (!shield0.begin()) {
    Serial.println(" Smotor shield init failed");
    delay(1000);
  }
  ina240_init();
  // === INA219
  //ina219_init();
  Serial.println(" Swait 1 second...");
  delay(1000);
  Serial.println(" Sready");
}

bool allEqual(bool a[], bool b[], size_t n) {
  for (size_t i = 0; i < n; i ++)
    if (a[i] != b[i]) return false;
  return true;
}

void loop() {
  static unsigned long prev = 0;
  static bool prev_values[4] = { 0 };
  unsigned long now = micros();
  ina240_update();
  
  // Protect against drift (?):
  // - A is shorted with B using a motor train
  // - CAAN000 or CABN000
  // - try CBAN020 and CBBN020:
  //   - CBAN020: 508 (-03)
  //   - CBBN020: 550 (+39)
  // - ignore ina240_values when pwm applied is 0.
  for (size_t i = 0; i < 4; i ++) {
    Line line;
    if (i == 0) line = lineA;
    if (i == 1) line = lineB;
    if (i == 2) line = lineC;
    if (i == 3) line = lineD;
    if (line.pwm == 0)
      ina240_values[i] = false;
  }

  if (prev + 3000 <= now) {
    if (!allEqual(prev_values, ina240_values, sizeof(ina240_values)/sizeof(ina240_values[0]))) {
      if (debugPoint) {
        //show(0, A) show(1, B) show(2, C) show(3, D)
        //Serial.print(",thresholdPositive:");
        //Serial.print(ina219_threshold);
        //Serial.print(",thresholdNegative:");
        //Serial.print(-ina219_threshold);
        //Serial.println();
      }
      Serial.print(" DC");
      Serial.print("A");
      Serial.print(ina240_values[0]);
      Serial.print("B");
      Serial.print(ina240_values[1]);
      Serial.print("C");
      Serial.print(ina240_values[2]);
      Serial.print("D");
      Serial.print(ina240_values[3]);
      Serial.print("T");
      Serial.println(now);
    }
    prev = now;
    for (size_t i = 0; i < sizeof(ina240_values)/sizeof(ina240_values[0]); i ++)
      prev_values[i] = ina240_values[i];
  }
  Line_update(&lineA);
  Line_update(&lineB);
  Line_update(&lineC);
  Line_update(&lineD);
  handleSLCP();
}

void handleShort(bool isShort) {
  static char digitBuffer[4];
  // TODO: error checking
  int line = Serial.read();
  Line *t;
  if (line == 'A')
    t = &lineA;
  else if (line == 'B')
    t = &lineB;
  else if (line == 'C')
    t = &lineC;
  else if (line == 'D')
    t = &lineD;
  else {
    Serial.print(" Einvalid line ");
    Serial.println(line);
    return;
  }
  int direction = Serial.read();
  int brake = Serial.read();
  size_t read = Serial.readBytes(digitBuffer, 3);
  if (read != 3) {
    Serial.println(" Eread power: not enough chars");
    return;
  }
  int speed = atoi(digitBuffer);
  if (speed < 0 || speed > 255) {
    Serial.println(" Eout of range");
    return;
  };
  if (isShort) {
    char t_ = Serial.read();
    if (t_ != 'T') {
      Serial.print(" Eread short: literal 'T' expected, got ");
      Serial.println(t_, HEX);
      return;
    }
    char buffer[6];
    size_t read = Serial.readBytes(buffer, 5);
    if (read != 5) {
      Serial.println(" Eread duration: not enough chars");
      return;
    }
    buffer[5] = '\0';
    int duration = atoi(buffer);
    t->stop_ms = millis() + duration;
    t->stop_brake = Serial.read() == 'Y';
  }
  int eol = Serial.read();
  if (eol != '\n') {
    Serial.println(" Eexpected eol");
    return;
  }

  if (direction == 'A')
    Line_setDirection(t, true);
  else if (direction == 'B')
    Line_setDirection(t, false);
  Line_setPwm(t, speed, brake == 'Y');
  Serial.println(" Ook");
  return;
}

// handle in/out of soyuu line control protocol
void handleSLCP() {
  static char buffer[11];
  if (Serial.available() == 0)
    return;
  int kind = Serial.read();
  if (kind == 'I') {
    Serial.print(" Isoyuu-line/" VARIANT "/");
    Serial.println(instance);
    int eol = Serial.read();
    if (eol != '\n') {
      Serial.println(" Eexpected eol");
    }
  } else if (kind == 'S') {
    handleShort(true);
  } else if (kind == 'C') {
    handleShort(false);
  } else if (kind == 'L') {
    buffer[0] = Serial.read();
    buffer[1] = Serial.read();
    buffer[2] = Serial.read();
    buffer[3] = Serial.read();
    buffer[4] = Serial.read();
    buffer[5] = '\0';
    ina240_offset = atoi(buffer);
    Serial.print("ina240_offset set to ");
    Serial.print(ina240_offset);
    Serial.println(". Note: this is only saved to RAM.");
  } else if (kind == 'M') {
    buffer[0] = Serial.read();
    buffer[1] = Serial.read();
    buffer[2] = Serial.read();
    buffer[3] = Serial.read();
    buffer[4] = Serial.read();
    buffer[5] = '\0';
    ina240_threshold = atoi(buffer);
    Serial.print("ina240_threshold set to ");
    Serial.print(ina240_threshold);
    Serial.println(". Note: this is only saved to RAM.");
  } else if (kind == 'f') {
    buffer[0] = Serial.read();
    buffer[1] = Serial.read();
    buffer[2] = Serial.read();
    buffer[3] = '\0';
    ina240_hysteresis_delay_us = (long) atoi(buffer) * 1000;
    Serial.print("ina240_hysteresis_delay_us set to ");
    Serial.print(ina240_hysteresis_delay_us);
    Serial.println(". Note: this is only saved to RAM.");
  } else if (kind == 'G') {
    debug = !debug;
  } else if (kind == 'g') {
    debugPoint = !debugPoint;
  } else if (kind == 'H') {
    Serial.println("Clearing EEPROM...");
    for (int i = 0; i < EEPROM.length(); i++) {
      EEPROM.update(i, 0);
    }
    Serial.println("Cleared EEPROM.");
  } else if (kind == 'K') {
    // set instance (the 1 part of v1/1)
    static char newInstance[0x21] = {0};
    size_t n = Serial.readBytes(newInstance, 0x20);
    for (size_t i = 0; i < 0x20; i++) {
      if (newInstance[i] == '\n') {
        newInstance[i] = '\0';
      }
    }
    for (size_t i = 0; i < 0x20; i++) {
      EEPROM.update(EEPROM_INSTANCE_ADDR + i, newInstance[i]);
    }
    Serial.println("Wrote instance name to EEPROM.");
    Serial.print("Old instance name: ");
    Serial.println(instance);
    Serial.print("New instance name: ");
    Serial.println(newInstance);
    for (size_t i = 0; i < 0x20; i++) {
      instance[i] = newInstance[i];
    }
  } else if (kind == '\n') {
    // ignore
  } else {
    Serial.print(" Eunknown kind ");
    Serial.println(kind);
  }
}
