unsigned long logInterval = 1000;
bool logData = true;

void setup() {
  Serial.begin(9600);
  pinMode(7, INPUT_PULLUP);
  pinMode(10, INPUT_PULLUP);
  Serial.println(" Sready");
}

void loop() {
  handleSLCP();
  if (logData)
    doLogging();
}

void doLogging() {
  static unsigned long prev = millis();
  unsigned long now = millis();
  if (now - prev <= logInterval)
    return;
  prev = now;
  Serial.print(" DA");
  Serial.print(digitalRead(7));
  Serial.print("B");
  Serial.print(digitalRead(10));
  Serial.print("T");
  Serial.println(now);
}

void handleSLCP() {
  static char buffer[11];
  if (Serial.available() == 0)
    return;
  int kind = Serial.read();
  if (kind == 'I') {
    Serial.println(" Isoyuu-breakbeam itsybitsy0/0");
    int eol = Serial.read();
    if (eol != '\n') {
      Serial.println(" Eexpected eol");
    }
  } else if (kind == 'S') {
    size_t read = Serial.readBytesUntil('\n', buffer, 10);
    // [1] op - only 'i' for interval
    // [4] val - for 'i', 4 digits on decimal for log interval ms
    if (read != 5) {
      Serial.print(" Einvalid length ");
      Serial.println(read);
      return;
    }
    if (buffer[0] != 'i') {
      Serial.print(" Einvalid op ");
      Serial.println(buffer[0]);
      return;
    }
    int val = atoi(buffer + 1);
    logInterval = (unsigned long)val;
    Serial.print(" Olog interval changed to ");
    Serial.println(logInterval);
  } else if (kind == 'L') {
    size_t read = Serial.readBytesUntil('\n', buffer, 10);
    if (read != 1) {
      Serial.print(" Einvalid length ");
      Serial.println(read);
      return;
    }
    if (buffer[0] != '0' && buffer[0] != '1') {
      Serial.print(" Einvalid op ");
      Serial.println(buffer[0]);
      return;
    }
    logData = buffer[0] == '1';
    Serial.print(" Ologging changed to ");
    Serial.println(logData);
  } else {
    Serial.print(" Eunknown kind ");
    Serial.println(kind);
  }
}
