#include <Adafruit_INA219.h>
#include <EEPROM.h>

bool debug = false;

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

int ina219_weight = 92;
float ina219_elapsed_weight = 1.0;
int ina219_threshold = 2000;
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

static void ina219_update_single(int i, Adafruit_INA219 *ina219, int elapsed,
                                 bool handle) {
  long current_uA = ina219->getCurrent_mA() * 1000;
  int weight = ina219_weight + (int)(ina219_elapsed_weight * (float)elapsed);
  // TODO: ina219 moving average is highly affected my timing
  ina219_lines[i].direct_uA = current_uA;
  if (handle) {
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
    ina219_lines[i].direct_uA -= calibs[i].offset_uA;
  }
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
      (ina219_lines[i].weighted_uA * weight + current_uA * (100 - weight)) /
      100;
}

void ina219_update(int elapsed, bool handle) {
  ina219_update_single(0, &ina2190, elapsed, handle);
  ina219_update_single(1, &ina2191, elapsed, handle);
  ina219_update_single(2, &ina2192, elapsed, handle);
  ina219_update_single(3, &ina2193, elapsed, handle);
}

void ina219_load_calibrate() {
  for (int i = 0; i < 3; i ++) {
    EEPROM.get(EEPROM_CALIBRATION_ADDR + i * 4, calibs[i].offset_uA);
    Serial.print(i);
    Serial.print(" offset: ");
    Serial.println(calibs[i].offset_uA);
  }
}

void ina219_calibrate() {
  long cums[4] = {0};
  const unsigned long timeframe = 40000000;
  Serial.print("Timeframe: ");
  Serial.print(timeframe);
  Serial.println("µs");
  unsigned long prev = 0;
  unsigned long now = micros();
  unsigned long start = now;
  long count = 0;
  while (now - start < timeframe) {
    now = micros();
    if (prev + 3000 <= now) {
      ina219_update((now - prev) / 1000, false);
      count++;
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
  for (int i = 0; i < 3; i ++) {
    cums[i] += ina219_lines[i].weighted_uA;
  }
      prev = now;
    }
  }
  for (int i = 0; i < 3; i ++) {
    calibs[i].offset_uA = cums[i] / count;
  }
  for (int i = 0; i < 3; i ++) {
    Serial.print(i);
    Serial.print(" offset: ");
    Serial.print(calibs[i].offset_uA);
    Serial.print(" cum: ");
    Serial.println(cums[i]);
  }
          Serial.print("count: ");
  Serial.println(count);
  for (int i = 0; i < 3; i ++) {
    EEPROM.put(EEPROM_CALIBRATION_ADDR + i * 4, calibs[i].offset_uA);
  }
}

#undef INA219_LENGTH
