#include <EEPROM.h>

#define INA240_PIN0 A0
#define INA240_PIN1 A1
#define INA240_PIN2 A2
#define INA240_PIN3 A3

bool debug = false;
bool debugPoint = false;

struct ina240_line {
  int pin;
  unsigned long onUntil_us;
};

long ina240_shunt_ohm = 4;
long ina240_gain = 20;
struct ina240_line ina240_lines[4] = {
  { .pin = INA240_PIN0 },
  { .pin = INA240_PIN1 },
  { .pin = INA240_PIN2 },
  { .pin = INA240_PIN3 },
};
bool ina240_values[4] = { 0 };

int ina240_offset = -511;
int ina240_threshold = 8;
long ina240_hysteresis_delay_us = 100 * 1000;

void ina240_init() {
  for (int i = 0; i < 4; i ++) {
    pinMode(ina240_lines[i].pin, INPUT);
  }
}

static void ina240_update_single(int i) {
  int raw = analogRead(ina240_lines[i].pin);
  bool value = abs(raw+ina240_offset) > ina240_threshold;
  unsigned long now = micros();
  ina240_values[i] = (now > ina240_lines[i].onUntil_us) ? value : true;
  if (value) {
    ina240_lines[i].onUntil_us = now + ina240_hysteresis_delay_us;
  }
  if (debug) {
    Serial.print("raw");
    Serial.print(i);
    Serial.print(" ");
    Serial.print(raw);
    Serial.print(" value_raw");
    Serial.print(value);
    Serial.print(" ");
    Serial.print(raw);
    Serial.print(" value");
    Serial.print(ina240_values[i]);
    Serial.print(" ");
    Serial.println(raw);
  }
  return;
}

void ina240_update() {
  ina240_update_single(0);
  ina240_update_single(1);
  ina240_update_single(2);
  ina240_update_single(3);
  delay(100);
}

