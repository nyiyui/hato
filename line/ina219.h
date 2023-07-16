#include <Adafruit_INA219.h>
#include <EEPROM.h>
#include <Wire.h>

bool debug = false;
bool debugPoint = false;

struct ina219_calibration {
  long offset_uA;
};

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

static struct ina219_calibration calibs[4] = {0};
static bool use_calibs = true;
unsigned long ina219_count = 0;

int ina219_weight = 90;
float ina219_elapsed_weight = 1.0;
int ina219_threshold = 12000;
// Set this to a "high enough" threshold such that drift (due to high common-mode voltage) won't affect this - this drift is usually around 3 to 4 mA.
// https://e2e.ti.com/support/amplifiers-group/amplifiers/f/amplifiers-forum/790103/ina219-ina219---bidirectional-current-measurement-to-measure-motor-current?ReplyFilter=Answers&ReplySortBy=Answers&ReplySortOrder=Descending
// See issue #14 for details.

void ina219_init() {
  Wire.setWireTimeout();
  while (!ina2190.begin())
    Serial.println(" Sina2190 init failed"), delay(1000);
  while (!ina2191.begin())
    Serial.println(" Sina2191 init failed"), delay(1000);
  while (!ina2192.begin())
    Serial.println(" Sina2192 init failed"), delay(1000);
  while (!ina2193.begin())
    Serial.println(" Sina2193 init failed"), delay(1000);
  ina2190.setCalibration_16V_400mA();
  ina2191.setCalibration_16V_400mA();
  ina2192.setCalibration_16V_400mA();
  ina2193.setCalibration_16V_400mA();
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
  int weight = ina219_weight + (int)(ina219_elapsed_weight * (float)elapsed);
  // TODO: ina219 moving average is highly affected my timing
  ina219_lines[i].direct_uA = current_uA;
#define handle2(j)                                                             \
  do {if (i == j) {                                                                \
    Serial.print(#j "direct:");\
    Serial.print(ina219_lines[j].direct_uA);\
    Serial.print(","#j "result:");\
    Serial.println(ina219_lines[j].direct_uA-calib##j.offset_uA);\
  }} while (0)
  if (debug) {
    Serial.print(i);
    Serial.print("original");
    Serial.print(":");
    Serial.print(ina219_lines[i].direct_uA);
    Serial.print(",");
    Serial.print(i);
    Serial.print("result");
    Serial.print(":");
    Serial.print(ina219_lines[i].direct_uA-calibs[i].offset_uA);
    Serial.print(",");
  }
#undef handle2
  if (use_calibs) {
    ina219_lines[i].direct_uA -= calibs[i].offset_uA;
  }
  if (abs(current_uA) > abs(ina219_lines[i].weighted_uA)) {
    weight = 0;
  }
  // ignore as this is probably the non-duty-cycle part of PWM
  ina219_lines[i].weighted_uA =
      (ina219_lines[i].weighted_uA * weight + current_uA * (100 - weight)) /
      100;
}

void ina219_update(int elapsed) {
  ina219_update_single(0, &ina2190, elapsed);
  ina219_update_single(1, &ina2191, elapsed);
  ina219_update_single(2, &ina2192, elapsed);
  ina219_update_single(3, &ina2193, elapsed);
  ina219_count ++;
}

void ina219_load_calibrate() {
  for (int i = 0; i < 3; i ++) {
    EEPROM.get(EEPROM_CALIBRATION_ADDR + i * 4, calibs[i].offset_uA);
    Serial.print(i);
    Serial.print(" offset: ");
    Serial.println(calibs[i].offset_uA);
  }
}

static long cums[4] = {0};
static unsigned long calibrate_end = 0;

void ina219_calibrate_start(unsigned long duration) {
  for (int i = 0; i < 3; i ++) {
    cums[i] = 0;
  }
  ina219_count = 0;
  calibrate_end = millis() + duration;
}

bool ina219_calibrate_step_stop() {
  for (int i = 0; i < 3; i ++) {
    cums[i] += ina219_lines[i].weighted_uA;
  }
  if (millis() < calibrate_end) {
    return false;
  }
  for (int i = 0; i < 3; i ++) {
    calibs[i].offset_uA = cums[i] / ina219_count;
  }
  for (int i = 0; i < 3; i ++) {
    Serial.print(i);
    Serial.print(" offset: ");
    Serial.print(calibs[i].offset_uA);
    Serial.print(" cum: ");
    Serial.println(cums[i]);
  }
  Serial.print("ina219_count: ");
  Serial.println(ina219_count);
  for (int i = 0; i < 3; i ++) {
    EEPROM.put(EEPROM_CALIBRATION_ADDR + i * 4, calibs[i].offset_uA);
  }
  return true;
}

#undef INA219_LENGTH
