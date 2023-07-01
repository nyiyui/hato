#include <Adafruit_INA219.h>

struct ina219_line {
  long weighted_uA;
  long direct_uA;
  long underThresholdElapsed;
  bool ignoreUnder;
};

Adafruit_INA219 ina2190;
Adafruit_INA219 ina2191(0x41);
Adafruit_INA219 ina2192(0x44);
Adafruit_INA219 ina2193(0x45);
struct ina219_line ina219_lines[4] = {0};
#define INA219_LENGTH 4

int ina219_weight = 92;
float ina219_elapsed_weight = 1.0;
int ina219_threshold = 1000;
int ina219_uteThreshold = 50000;

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

static void ina219_update_single(int i, Adafruit_INA219 *ina219, int elapsed) {
  long current_uA = ina219->getCurrent_mA() * 1000;
  int weight = ina219_weight + (int) (ina219_elapsed_weight * (float) elapsed);
  // TODO: ina219 moving average is highly affected my timing
  ina219_lines[i].direct_uA = current_uA;
  if (abs(current_uA) < ina219_threshold) {
    ina219_lines[i].underThresholdElapsed += elapsed;
  }
  if (ina219_lines[i].underThresholdElapsed > ina219_uteThreshold) {
    ina219_lines[i].ignoreUnder = false;
  }
  if (abs(current_uA) >= ina219_threshold) {
    ina219_lines[i].ignoreUnder = true;
  }
  if (ina219_lines[i].ignoreUnder && abs(current_uA) < ina219_threshold) {
    return;
  }
  // ignore as this is probably the non-duty-cycle part of PWM
  ina219_lines[i].weighted_uA =
      (ina219_lines[i].weighted_uA * weight +
       current_uA * (100 - weight)) /
      100;
}

void ina219_update(int elapsed) {
  ina219_update_single(0, &ina2190, elapsed);
  ina219_update_single(1, &ina2191, elapsed);
  ina219_update_single(2, &ina2192, elapsed);
  ina219_update_single(3, &ina2193, elapsed);
}

#undef INA219_LENGTH