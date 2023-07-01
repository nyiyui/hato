#define DEBUG
#include "ina219.h"
#include <Adafruit_MotorShield.h>

typedef struct Line {
  Adafruit_DCMotor *motor;
  bool direction;
} line;

Adafruit_MotorShield shield0 = Adafruit_MotorShield();
bool debug = false;

static Line lineA = {
    .motor = shield0.getMotor(1),
    .direction = true,
};
static Line lineB = {
    .motor = shield0.getMotor(2),
    .direction = true,
};
static Line lineC = {
    .motor = shield0.getMotor(3),
    .direction = true,
};
static Line lineD = {
    .motor = shield0.getMotor(4),
    .direction = true,
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

void setup() {
  Serial.println(" Sstart");
  // === Motor Shield
  while (!shield0.begin()) {
    Serial.println(" Smotor shield init failed");
    delay(1000);
  }
  // === INA219
  ina219_init();
  Serial.println(" Swait 5 seconds...");
  delay(5000);
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
#ifdef DEBUG
    if (debug) {
      Serial.print("elapsed:");
      Serial.print(now - prev);
#define show(i, letter)                                                        \
  Serial.print(",w" #letter ":");                                              \
  Serial.print(ina219_lines[i].weighted_uA);                                   \
  Serial.print(",d" #letter ":");                                              \
  Serial.print(ina219_lines[i].direct_uA);
      show(0, A) show(1, B) show(2, C) show(3, D)
#undef show
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
  handleSLCP();
}

void handleShort(bool isShort) {
  static char digitBuffer[4];
  Serial.println(" Psc");
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
  int read = Serial.readBytes(digitBuffer, 3);
  if (read != 3) {
    Serial.println(" Enot enough");
    return;
  }
  int speed = atoi(digitBuffer);
  if (speed < 0 || speed > 255) {
    Serial.println(" Eout of range");
    return;
  };
  int eol = Serial.read();
  if (eol != '\n') {
    Serial.println(" Eexpected eol");
    return;
  }
  Serial.print(" D");
  Serial.print(direction);
  Serial.print(" B");
  Serial.print(brake);
  Serial.print(" S");
  Serial.print(speed);
  Serial.println(".");

  Serial.println(" Pstarting");
  if (direction == 'A')
    Line_setDirection(t, true);
  else if (direction == 'B')
    Line_setDirection(t, false);
  Line_setPwm(t, speed, brake == 'Y');
  if (isShort) {
    Serial.println(" Pwaiting");
    delay(100);
    Serial.println(" Pstopping");
    Line_setPwm(t, 0, true);
  }
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
    Serial.println(" Isoyuu-line-mega-0");
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
  } else {
    Serial.print(" Eunknown kind ");
    Serial.println(kind);
  }
}