bool ina240_debug = false;
int ina240_hysteresis_delay_ms = 100;
int ina240_offset = -511;
int ina240_threshold = 8;

struct channel {
  char name;

  int pwm_pin;
  int dir_pin;
  int sensor_pin;
  bool flip;

  unsigned long stop_ms;

  int val;

  bool tf_now;
  unsigned long _hys_until;

  int _prev_power;
};

void channel_setup(struct channel *c) {
  pinMode(c->pwm_pin, OUTPUT);
  pinMode(c->dir_pin, OUTPUT);
  if (c->sensor_pin >= 0)
    pinMode(c->sensor_pin, INPUT);
}

int signum(int x) {
  if (x == 0) return 0;
  if (x < 0) return -1;
  if (x > 0) return 1;
}

void channel_write(struct channel *c, int power) {
  //Serial.print("name ");
  //Serial.println(c->name);
  //Serial.print("power ");
  //Serial.println(power);
  //Serial.print("abs(power) ");
  //Serial.println(abs(power));
  //Serial.print("signum(power) ");
  //Serial.println(signum(power));
  //Serial.print("signum(c->_prev_power) ");
  //Serial.println(signum(c->_prev_power));
  //Serial.print("digitalWrite ");
  //Serial.println(power > 0 ? "HIGH" : "LOW");
  //Serial.print("analogWrite ");
  //Serial.println(abs(power));
  if (power != 0 && signum(power) != signum(c->_prev_power)) {
    digitalWrite(c->pwm_pin, 0); // prevent a possible moment of power in the wrong direction here
    digitalWrite(c->dir_pin, power > 0 != c->flip ? HIGH : LOW);
  }
  if (abs(c->_prev_power) != abs(power))
    analogWrite(c->pwm_pin, abs(power));
  c->_prev_power = power;
}

//#define channels_len 8
//struct channel channels[channels_len] = {
//  { .name = 'A', .pwm_pin = 3,  .dir_pin = 22, .sensor_pin = A8, },
//  { .name = 'B', .pwm_pin = 2,  .dir_pin = 18, .sensor_pin = A1, },
//  { .name = 'C', .pwm_pin = 7,  .dir_pin = 23, .sensor_pin = A2, },
//  { .name = 'D', .pwm_pin = 8,  .dir_pin = 24, .sensor_pin = A3, },
//  { .name = 'E', .pwm_pin = 9,  .dir_pin = 25, .sensor_pin = A4, },
//  { .name = 'F', .pwm_pin = 10, .dir_pin = 26, .sensor_pin = A5, },
//  { .name = 'G', .pwm_pin = 11, .dir_pin = 27, .sensor_pin = A6, },
//  { .name = 'H', .pwm_pin = 12, .dir_pin = 28, .sensor_pin = A7, },
//};
#define channels_len 16
struct channel channels[channels_len] = {
  { .name = 'A', .pwm_pin = 3,  .dir_pin = 22, .sensor_pin = A8,  .flip = true,  },
  { .name = 'B', .pwm_pin = 2,  .dir_pin = 23, .sensor_pin = A1,  .flip = true,  },
  { .name = 'C', .pwm_pin = 7,  .dir_pin = 24, .sensor_pin = A2,  .flip = true,  },
  { .name = 'D', .pwm_pin = 8,  .dir_pin = 25, .sensor_pin = A3,  .flip = false, }, // seems like the phase is different
  { .name = 'E', .pwm_pin = 9,  .dir_pin = 26, .sensor_pin = A4,  .flip = true,  },
  { .name = 'F', .pwm_pin = 10, .dir_pin = 27, .sensor_pin = A5,  .flip = true,  },
  { .name = 'G', .pwm_pin = 11, .dir_pin = 28, .sensor_pin = A6,  .flip = true,  },
  { .name = 'H', .pwm_pin = 12, .dir_pin = 29, .sensor_pin = A7,  .flip = true,  },
  { .name = 'I', .pwm_pin = 44, .dir_pin = 30, .sensor_pin = A9,  .flip = true,  },
  { .name = 'J', .pwm_pin = 45, .dir_pin = 31, .sensor_pin = A10, .flip = true,  },
  { .name = 'K', .pwm_pin = 46, .dir_pin = 32, .sensor_pin = A11, .flip = true,  },
  { .name = 'L', .pwm_pin = 6,  .dir_pin = 33, .sensor_pin = A12, .flip = true,  }, // higher-than-expected duty cycles
  { .name = 'M', .pwm_pin = 5,  .dir_pin = 34, .sensor_pin = A13, .flip = false, }, // higher-than-expected duty cycles
  { .name = 'N', .pwm_pin = 4,  .dir_pin = 35, .sensor_pin = A14, .flip = true,  }, // 980 Hz
  { .name = 'O', .pwm_pin = 13, .dir_pin = 36, .sensor_pin = A15, .flip = true,  }, // 980 Hz
  { .name = 'P', .pwm_pin = 40, .dir_pin = 37, .sensor_pin = -1,  .flip = true,  }, // no PWM
};

void channels_setup() {
  for (int i = 0; i < channels_len; i ++) {
    channel_setup(&channels[i]);
  }
  Serial.println("channels: setup done.");
}

void channels_updateSensors() {
  for (int i = 0; i < channels_len; i ++) {
    struct channel *c = &channels[i];
    unsigned long now = millis();
    int val;
    if (c->_prev_power == 0 || c->sensor_pin < 0)
      val = -ina240_offset;
    else
      val = analogRead(c->sensor_pin);
    val = abs(val);
    //Serial.print(val);
    bool tf_now = abs(val + ina240_offset) > ina240_threshold;
    if (tf_now) {
      c->_hys_until = now + ina240_hysteresis_delay_ms;
    }
    if (now <= c->_hys_until) {
      tf_now = true;
    }
    c->val = val;
    c->tf_now = tf_now;
    if (ina240_debug) {
      Serial.print(i);
      Serial.print(" val");
      Serial.print(val);
      Serial.print(" tf_now");
      Serial.println(tf_now);
    }
  }
}

void channels_sendDelta() {
  static bool values[channels_len] = {0};
  bool changed = false;
  for (int i = 0; i < channels_len; i ++) {
    struct channel *c = &channels[i];
    if (c->tf_now != values[i]) changed = true;
    values[i] = c->tf_now;
  }

  if (ina240_debug) {
    Serial.print("values ");
    for (int i = 0; i < channels_len; i ++) {
      Serial.print(values[i]);
    }
    Serial.println();
  }

  //if (!changed && !ina240_debug) return;
  if (changed) {
    Serial.print(" DC");
    for (int i = 0; i < channels_len; i ++) {
      struct channel c = channels[i];
      Serial.print((char) ('A'+i));
      Serial.print(c.tf_now);
    }
    Serial.print("T");
    Serial.println(millis());
  }
}

void channels_stop_update() {
  unsigned long now = millis();
  for (int i = 0; i < channels_len; i ++) {
    struct channel *c = &channels[i];
    if (0 != c->stop_ms && now > c->stop_ms) {
      channel_write(c, 0);
      Serial.print(" DSL");
      Serial.print((char) ('A'+i));
      Serial.print("T");
      Serial.println(now);
      c->stop_ms = 0;
    }
  }
}
