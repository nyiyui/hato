#include "channels.h"
#include <EEPROM.h>

#define VARIANT "v4"
// Instance is initially set to this, if nothing exists at EEPROM_INSTANCE_ADDR.
#define INSTANCE "unset"

// === EEPROM Layout
// 00-10  variant string
// 10-31  instance name (null-terminated)
#define EEPROM_VERSION_ADDR 0x0
#define EEPROM_INSTANCE_ADDR 0x10

static char instance[0x21] = INSTANCE;

void setup() {
  Serial.begin(9600);
  Serial.println("start");
  if (EEPROM.read(EEPROM_INSTANCE_ADDR) == 0) {
    Serial.println("Initialising EEPROM with instance name...");
    static char *newInstance = instance;
    for (size_t i = 0; i < 0x20; i++) {
      EEPROM.update(EEPROM_INSTANCE_ADDR + i,
                    i < strlen(newInstance) ? newInstance[i] : 0);
    }
    Serial.println("Initialised EEPROM with instance name.");
  } else {
    for (size_t i = 0; i < 0x20; i++) {
      instance[i] = EEPROM.read(EEPROM_INSTANCE_ADDR + i);
    }
  }
  channels_setup();
  Serial.println("wait 1 second...");
  delay(1000);
  Serial.println("ready");
}

void handleSLCP();

void loop() {
  channels_updateSensors();
  channels_sendDelta();
  channels_stop_update();
  handleSLCP();
}

void handleShort(bool isShort) {
  // TODO: error checking
  size_t i = Serial.read()-'A';
  struct channel *c = &channels[i];
  if (i < 0 || i >= channels_len) {
    Serial.print(" Einvalid line ");
    Serial.println('A'+i);
    return;
  }
  int direction = Serial.read();
  Serial.read(); // read and dispose brake (not supported on kdss/v4)
  int power;
  {
    static char buffer[4];
    size_t read = Serial.readBytes(buffer, 3);
    if (read != 3) {
      Serial.println(" Eread power: not enough chars");
      return;
    }
    power = atoi(buffer);
  }
  if (power < 0 || power > 255) {
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
    c->stop_ms = millis() + duration;
    Serial.read(); // read and dispose brake (not supported on kdss/v4)
  }
  int eol = Serial.read();
  if (eol != '\n') {
    Serial.println(" Eexpected eol");
    return;
  }

  if (direction == 'B')
    power = -power;
  channel_write(c, power);
  Serial.print(" Owrite ");
  Serial.print(power);
  Serial.print(" to ");
  Serial.print('A'+i);
  Serial.println(".");
  return;
}

// handle in/out of soyuu line control protocol
void handleSLCP() {
  static char buffer[11];
  if (Serial.available() == 0)
    return;
  int kind = Serial.read();
  if (kind == 'I') {
    Serial.print(" Isoyuu-kdss/" VARIANT "/");
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
    ina240_hysteresis_delay_ms = (long) atoi(buffer);
    Serial.print("ina240_hysteresis_delay_ms set to ");
    Serial.print(ina240_hysteresis_delay_ms);
    Serial.println(". Note: this is only saved to RAM.");
  } else if (kind == 'G') {
    ina240_debug = !ina240_debug;
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
