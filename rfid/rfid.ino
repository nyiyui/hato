void setup() {
  Serial.begin(9600);
  pinMode(7, INPUT_PULLUP);
  pinMode(10, INPUT_PULLUP);
  Serial.println(" Sready");
}

void loop() {
  handleSLCP();
  doLogging();
}

#define CLAMP_LIMIT 300
#define THRESHOLD 290

long clamp(long a) {
  if (a < 0) return 0;
  if (a > CLAMP_LIMIT) return CLAMP_LIMIT;
  return a;
}

long update(bool current, long weighted, unsigned long elapsed) {
  if (current) weighted += (long) elapsed;
  if (!current) weighted -= (long) elapsed;
  return clamp(weighted);
}

void doLogging() {
  static long aWeighted = 0;
  static long bWeighted = 0;
  static bool aOld = false;
  static bool bOld = false;
  static unsigned long prev = millis();
  unsigned long now = millis();
  unsigned long elapsed = now - prev;
  //          ( types? wat r those )
  //          |/
  // ¯\_(ツ)_/¯
  bool aRaw = digitalRead(7) == HIGH;
  bool bRaw = digitalRead(10) == HIGH;
  aWeighted = update(aRaw, aWeighted, elapsed);
  bWeighted = update(bRaw, bWeighted, elapsed);
  /*
  Serial.print(" aWeighted ");
  Serial.print(aWeighted);
  Serial.print(" bWeighted ");
  Serial.print(bWeighted);
  Serial.print(" elapsed ");
  Serial.print(elapsed);
  Serial.println();
  */
  bool aCur = aWeighted > THRESHOLD;
  bool bCur = bWeighted > THRESHOLD;
  if (aCur != aOld || bCur != bOld) {
    Serial.print(" DA");
    Serial.print(aCur);
    Serial.print("B");
    Serial.print(bCur);
    Serial.print("T");
    Serial.println(now);
  }
  aOld = aCur;
  bOld = bCur;
  prev = now;
}

void handleSLCP() {
  static char buffer[11];
  if (Serial.available() == 0)
    return;
  int kind = Serial.read();
  if (kind == 'I') {
    Serial.println(" Isoyuu-breakbeam/itsybitsy0/0");
    int eol = Serial.read();
    if (eol != '\n') {
      Serial.println(" Eexpected eol");
    }
  } else {
    Serial.print(" Eunknown kind ");
    Serial.println(kind);
  }
}
