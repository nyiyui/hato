// HATO breakbeam

struct sensor {
  char id; // identifying (unique) uppercase character
  int pin; // pin to use in digitalRead
  unsigned long pos; // one sensor must have a position of 0
};

static struct sensor sensors[] = {
  {'A', 7, 0},
  {'B', 10, 248000},
  {'C', 12, 496000}
  //{'A', 5, 0},
  //{'B', 10, 248000},
  //{'C', 9, 496000}
};

#define SENSORS_LENGTH sizeof(sensors) / sizeof(sensor)

#define VARIANT "itsybitsy0/0"

#define CLAMP_LIMIT 300 // max weight (in weighteds)
#define THRESHOLD 290 // threshold weight for on/off determination

void setup() {
  Serial.begin(9600);
  for (int i = 0; i < SENSORS_LENGTH; i++) {
    pinMode(sensors[i].pin, INPUT_PULLUP);
  }
  Serial.println(" Sready");
}

void loop() {
  handleSLCP();
  doLogging();
}

long clamp(long a) {
  if (a < 0)
    return 0;
  if (a > CLAMP_LIMIT)
    return CLAMP_LIMIT;
  return a;
}

long update(bool current, long weighted, unsigned long elapsed) {
  if (current)
    weighted += (long)elapsed;
  if (!current)
    weighted -= (long)elapsed;
  return clamp(weighted);
}

void doLogging() {
  static long weighteds[SENSORS_LENGTH] = {0};
  static bool curs[SENSORS_LENGTH] = {0};
  static bool olds[SENSORS_LENGTH] = {0};
  static unsigned long prev = millis();
  unsigned long now = millis();
  unsigned long elapsed = now - prev;
  bool changed = false;
  for (int i = 0; i < SENSORS_LENGTH; i++) {
    //          ( types? wat r those )
    //          |/
    // ¯\_(ツ)_/¯
    bool raw = digitalRead(sensors[i].pin) == HIGH;
    weighteds[i] = update(raw, weighteds[i], elapsed);
    curs[i] = weighteds[i] > THRESHOLD;
    changed |= curs[i] != olds[i];
#      ifdef DEBUG
    Serial.print(" debug");
    Serial.print(sensors[i].id);
    Serial.print(" w=");
    Serial.print(weighteds[i]);
    Serial.print(" e=");
    Serial.print(elapsed);
    Serial.print("\t");
#      endif
    olds[i] = curs[i];
  }
#  ifdef DEBUG
  Serial.println();
#  endif
  if (changed) {
    Serial.print(" D");
    for (int i = 0; i < SENSORS_LENGTH; i++) {
      Serial.print(sensors[i].id);
      Serial.print(curs[i]);
    }
    Serial.print("T");
    Serial.println(now);
  }
  prev = now;
}

void handleSLCP() {
  static char buffer[11];
  if (Serial.available() == 0)
    return;
  int kind = Serial.read();
  if (kind == 'I') {
    Serial.println(" Isoyuu-breakbeam/" VARIANT);
    int eol = Serial.read();
    if (eol != '\n') {
      Serial.println(" Eexpected eol");
    }
  } else if (kind == 'J') {
    int eol = Serial.read();
    if (eol != '\n') {
      Serial.println(" Eexpected eol");
    }
    Serial.print(" Isoyuu-breakbeam/" VARIANT ";");
    for (int i = 0; i < SENSORS_LENGTH; i++) {
      Serial.print("J");
      Serial.print(sensors[i].id);
      Serial.print("P");
      Serial.print(sensors[i].pos);
      if (i != SENSORS_LENGTH-1)
        Serial.print(" ");
    }
    Serial.println();
  } else {
    Serial.print(" Eunknown kind ");
    Serial.println(kind);
  }
}
