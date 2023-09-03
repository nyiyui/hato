#include <EEPROM.h>

bool debug = false;
bool debugPoint = false;

struct ina240_line {
  unsigned long onUntil_us;
  bool now;
};

long ina240_shunt_ohm = 4;
long ina240_gain = 20;
const int ina240_pins[4] = { INA240_PIN0, INA240_PIN1, INA240_PIN2, INA240_PIN3 };
struct ina240_line ina240_lines[4] = {0};
#define INA240_LENGTH 4

int ina240_threshold = 1000;
long ina240_hysteresis_delay_us = 100 * 1000;

void ina240_init() {
  for (int i = 0; i < 4; i ++) {
    pinMode(ina240_pins[i], INPUT);
  }
}

static void ina240_update_single(int i) {
  int raw = analogRead(ina240_pins[i]);
  Serial.print("raw ");
  Serial.println(current_uA);
  ina240_lines[i].now = abs(raw-511) > 10;
  return;
}

void ina240_update() {
  ina240_update_single(0);
  ina240_update_single(1);
  ina240_update_single(2);
  ina240_update_single(3);
  delay(100);
}

#undef INA240_LENGTH
