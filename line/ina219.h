#include <Adafruit_INA219.h>

struct ina219_line {
  long weighted_uA;
};

Adafruit_INA219 ina2190;
Adafruit_INA219 ina2191(0x41);
Adafruit_INA219 ina2192(0x44);
Adafruit_INA219 ina2193(0x45);
struct ina219_line ina219_lines[4] = { 0 };
#define INA219_LENGTH 4

int ina219_weight = 92;

void ina219_init() {
  while (!ina2190.begin())
    Serial.println(" Sina2190 init failed"), delay(1000);
  while (!ina2191.begin())
    Serial.println(" Sina2191 init failed"), delay(1000);
  while (!ina2192.begin())
    Serial.println(" Sina2192 init failed"), delay(1000);
  while (!ina2193.begin())
    Serial.println(" Sina2193 init failed"), delay(1000);
  ina2190.setCalibration_32V_1A();
  ina2191.setCalibration_32V_1A();
  ina2192.setCalibration_32V_1A();
  ina2193.setCalibration_32V_1A();
}

static const long CLAMP_LIMIT = 300;

static long clamp(long a) {
  if (a < 0)
    return 0;
  if (a > CLAMP_LIMIT)
    return CLAMP_LIMIT;
  return a;
}

static void ina219_update_single(int i, Adafruit_INA219 *ina219) {
  float current = ina219->getCurrent_mA();
  Serial.print("direct_uA:");
  Serial.print(current * 1000);
  Serial.print(",");
  ina219_lines[i].weighted_uA = (ina219_lines[i].weighted_uA * ina219_weight + (long) (current * 1000) * (100-ina219_weight))/100;
}

void ina219_update() {
  ina219_update_single(0, &ina2190);
  ina219_update_single(1, &ina2191);
  ina219_update_single(2, &ina2192);
  ina219_update_single(3, &ina2193);
}

#undef INA219_LENGTH
