bool ina240_debug = false;
int ina240_hysteresis_delay_ms = 50;
int ina240_offset = -511;
int ina240_threshold = 8;

struct channel {
  char name;

  int pwm_pin;
  int dir_pin;
  int sensor_pin;

  unsigned long stop_ms;

  int val;
  int _hys_val;
  unsigned long _hys_until;

  bool tf_now;
  bool tf_prev;

  int _prev_power;
};

void channel_setup(struct channel *c) {
  pinMode(c->pwm_pin, OUTPUT);
  pinMode(c->dir_pin, OUTPUT);
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
  if (power != 0 && signum(power) != signum(c->_prev_power))
    digitalWrite(c->dir_pin, power > 0 ? HIGH : LOW);
  if (abs(c->_prev_power) != abs(power))
    analogWrite(c->pwm_pin, abs(power));
  c->_prev_power = power;
}

void channel_updateSensor(struct channel *c) {
  unsigned long now = millis();
  int val;
  if (c->_prev_power == 0)
    val = 0;
  else
    val = analogRead(c->sensor_pin);
  val = abs(val);
  if (now > c->_hys_until || val >= c->_hys_val) {
    c->_hys_val = val;
    c->_hys_until = now + ina240_hysteresis_delay_ms;
  }
  if (now > c->_hys_until)
    c->val = val;
  else
    c->val = c->_hys_val;
  c->tf_prev = c->tf_now;
  c->tf_now = abs(c->val) + ina240_offset > ina240_threshold;
  if (ina240_debug) {
    Serial.print(c->name);
    Serial.print("val:");
    Serial.print(c->val);
    Serial.print(",");
    Serial.print(c->name);
    Serial.print("tf:");
    Serial.print(c->tf_now * 1000);
  }
}

#define channels_len 8
struct channel channels[channels_len] = {
  { .name = 'A', .pwm_pin = 2,  .dir_pin = 18, .sensor_pin = A0, },
  { .name = 'B', .pwm_pin = 3,  .dir_pin = 22, .sensor_pin = A1, },
  { .name = 'C', .pwm_pin = 7,  .dir_pin = 23, .sensor_pin = A2, },
  { .name = 'D', .pwm_pin = 8,  .dir_pin = 24, .sensor_pin = A3, },
  { .name = 'E', .pwm_pin = 9,  .dir_pin = 25, .sensor_pin = A4, },
  { .name = 'F', .pwm_pin = 10, .dir_pin = 26, .sensor_pin = A5, },
  { .name = 'G', .pwm_pin = 11, .dir_pin = 27, .sensor_pin = A6, },
  { .name = 'H', .pwm_pin = 12, .dir_pin = 28, .sensor_pin = A7, },
};

void channels_setup() {
  for (int i = 0; i < channels_len; i ++) {
    channel_setup(&channels[i]);
  }
  Serial.println("channels: setup done.");
}

void channels_updateSensors() {
  for (int i = 0; i < channels_len; i ++) {
    channel_updateSensor(&channels[i]);
  }
}

void channels_sendDelta() {
  bool changed = false;
  for (int i = 0; i < channels_len; i ++) {
    struct channel c = channels[i];
    if (c.tf_now != c.tf_prev) changed = true;
  }
  if (!changed) return;
  Serial.print(" DC");
  for (int i = 0; i < channels_len; i ++) {
    struct channel c = channels[i];
    Serial.print((char) ('A'+i));
    Serial.print(c.tf_now);
  }
  Serial.print("T");
  Serial.println(millis());
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
