#define DEBUG
#include <Adafruit_MotorShield.h>
#include <EEPROM.h>

#define VARIANT "v2"
// Instance is initially set to this, if nothing exists at EEPROM_INSTANCE_ADDR.
#define INSTANCE "4"

// === EEPROM Layout
// 00-10  version string
// 10-31  instance name (null-terminated)
// 40-TBD calibration data
#define EEPROM_VERSION_ADDR 0x0
#define EEPROM_INSTANCE_ADDR 0x10
#define EEPROM_CALIBRATION_ADDR 0x40
#include "ina219.h"

static bool calibrating = false;

typedef struct Line {
  char id;
  Adafruit_DCMotor *motor;
  bool direction;
  unsigned long stop_ms;
  bool stop_brake;
} line;

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
}

void Line_update(Line *line) {
  unsigned long now = millis();
  if (0 != line->stop_ms && now > line->stop_ms) {
    Line_setPwm(line, 0, line->stop_brake);
    // TODO: send confirmation
    Serial.print(" DCL");
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
  if (EEPROM.read(EEPROM_CALIBRATION_ADDR) != 0) {
    Serial.println(" SLoading calibration data from EEPROM...");
    ina219_load_calibrate();
    Serial.println(" SLoaded calibration data from EEPROM.");
  }
  // === Motor Shield
  while (!shield0.begin()) {
    Serial.println(" Smotor shield init failed");
    delay(1000);
  }
  // === INA219
  ina219_init();
  Serial.println(" Swait 1 second...");
  delay(1000);
  Serial.println(" Sready");
}

void loop() {
  static unsigned long prev = 0;
  static bool prevA = false;
  static bool prevB = false;
  static bool prevC = false;
  static bool prevD = false;
  unsigned long now = micros();
  if (prev + 3000 <= now) {
    ina219_update((now - prev) / 1000);
    if (calibrating) {
      calibrating = !ina219_calibrate_step_stop();
      if (!calibrating) {
        Serial.print("Done calibration.");
      }
    }
#ifdef DEBUG
    if (debug) {
      //Serial.print("elapsed:");
      //Serial.print(now - prev);
#define show(i, letter)                                                        \
  Serial.print(",w" #letter ":");                                              \
  Serial.print(ina219_lines[i].weighted_uA);                                   \
  Serial.print(",d" #letter ":");                                              \
  Serial.print(ina219_lines[i].direct_uA);
      show(0, A) show(1, B) show(2, C) show(3, D)
      Serial.print(",thresholdPositive:");
      Serial.print(ina219_threshold);
      Serial.print(",thresholdNegative:");
      Serial.print(-ina219_threshold);
      Serial.println();
    }
#endif
    bool nowA = abs(ina219_lines[0].weighted_uA) > ina219_threshold;
    bool nowB = abs(ina219_lines[1].weighted_uA) > ina219_threshold;
    bool nowC = abs(ina219_lines[2].weighted_uA) > ina219_threshold;
    bool nowD = abs(ina219_lines[3].weighted_uA) > ina219_threshold;
#define same(letter) now##letter == prev##letter
    if (!(same(A) && same(B) && same(C) && same(D))) {
#undef same
      if (debugPoint) {
        show(0, A) show(1, B) show(2, C) show(3, D)
        Serial.print(",thresholdPositive:");
        Serial.print(ina219_threshold);
        Serial.print(",thresholdNegative:");
        Serial.print(-ina219_threshold);
        Serial.println();
      }
#undef show
      Serial.print(" D");
      Serial.print("A");
      Serial.print(nowA);
      Serial.print("B");
      Serial.print(nowB);
      Serial.print("C");
      Serial.print(nowC);
      Serial.print("D");
      Serial.print(nowD);
      Serial.print("T");
      Serial.println(now);
    }
    prev = now;
    prevA = nowA;
    prevB = nowB;
    prevC = nowC;
    prevD = nowD;
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
  Serial.print(" PL");
  Serial.print(line);
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
  Serial.print(" D");
  Serial.print(direction);
  Serial.print(" B");
  Serial.print(brake);
  Serial.print(" S");
  Serial.print(speed);
  Serial.println(".");
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
  } else if (kind == 'D') {
    buffer[0] = Serial.read();
    buffer[1] = Serial.read();
    buffer[2] = Serial.read();
    buffer[3] = '\0';
    ina219_weight = atoi(buffer);
    Serial.print("ina219_weight set to ");
    Serial.print(ina219_weight);
    Serial.println(". Note: this is only saved to RAM.");
  } else if (kind == 'E') {
    buffer[0] = Serial.read();
    buffer[1] = Serial.read();
    buffer[2] = Serial.read();
    buffer[3] = Serial.read();
    buffer[4] = Serial.read();
    buffer[5] = '\0';
    ina219_elapsed_weight = atof(buffer);
    Serial.print("ina219_elapsed_weight set to ");
    Serial.print(ina219_elapsed_weight);
    Serial.println(". Note: this is only saved to RAM.");
  } else if (kind == 'F') {
    buffer[0] = Serial.read();
    buffer[1] = Serial.read();
    buffer[2] = Serial.read();
    buffer[3] = '\0';
    ina219_threshold = atoi(buffer);
    Serial.print("ina219_threshold set to ");
    Serial.print(ina219_threshold);
    Serial.println(". Note: this is only saved to RAM.");
  } else if (kind == 'G') {
    debug = !debug;
  } else if (kind == 'g') {
    debugPoint = !debugPoint;
  } else if (kind == 'H') {
    // calibration
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
  } else if (kind == 'L') {
    Serial.println("Starting calibration for 40s...");
    calibrating = true;
    ina219_calibrate_start(40000);
  } else if (kind == 'l') {
    char line = Serial.read();
    Serial.readBytes(buffer, 10);
    int i = line - 'A';
    calibs[i].offset_uA = atoi(buffer);
    EEPROM.put(EEPROM_CALIBRATION_ADDR + i * 4, calibs[i].offset_uA);
  } else if (kind == 'M') {
    ina219_load_calibrate();
  } else {
    Serial.print(" Eunknown kind ");
    Serial.println(kind);
  }
}
