#include <Adafruit_INA219.h>

struct ina219_line {
  int elapsed;
  long cleaned_uA;
#  ifdef DEBUG
  long direct_uA;
#  endif
};

Adafruit_INA219 ina2190;
Adafruit_INA219 ina2191(0x41);
Adafruit_INA219 ina2192(0x44);
Adafruit_INA219 ina2193(0x45);
struct ina219_line ina219_lines[4] = {0};
#define INA219_LENGTH 4

int ina219_lag = 30000;
int ina219_threshold = 2000; // measured on E233-3016 tail lamp

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
  // Assume spikes are from PWM, and as all we want to know is if there is a load or not, we can ignore the not-in-duty-cycle parts (non-spikes).
  // Assume spikes' max values are representative of the load. Also not detrimental if it's overestimating.
#  define line2 ina219_lines[i];
  ina219_line line = ina219_lines[i];
  line2->elapsed += elapsed;
  long current_uA;
  float current_mA = ina219->getCurrent_mA();
  current_uA = current_mA * 1000;
  line2->direct_uA = current_uA;
  if (line2->elapsed > ina219_lag || line.cleaned_uA < current_uA) {
    line2->cleaned_uA = current_uA;
    line2->elapsed = 0;
  }
}

void ina219_update(int elapsed) {
  ina219_update_single(0, &ina2190, elapsed);
  ina219_update_single(1, &ina2191, elapsed);
  ina219_update_single(2, &ina2192, elapsed);
  ina219_update_single(3, &ina2193, elapsed);
}

#undef INA219_LENGTH
