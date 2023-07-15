#define LINECALIB_RATIO 8
#define LINECALIB_LENGTH (256*2/LINECALIB_RATIO)

typedef struct LineCalib {
  int offsets[LINECALIB_LENGTH];
} LineCalib;

LineCalib lineCalibs[4] = {0};

static int currentLine = 0;
static bool currentDir = false;
static int currentPwm = 0;
static unsigned long next;
static unsigned long lastMeasurement = 0;
static long cum;
static bool linecalib_calibrating = false;

void linecalib_show() {
  for (size_t i = 0; i < 4; i ++) {
    Serial.print("Z: show line");
    Serial.println(i + 'A');
    for (size_t j = 0; j < LINECALIB_LENGTH; j ++) {
      Serial.print(lineCalibs[i].offsets[j]);
      if (j != LINECALIB_LENGTH-1) Serial.print(",");
    }
    Serial.println();
  }
}

void linecalib_start() {
  cum = 0;
  next = -1;
  currentLine = 0;
  currentDir = false;
  currentPwm = 0;
  linecalib_calibrating = true;
}

bool linecalib_step_stop() {
  if (!linecalib_calibrating) return false;
  unsigned long now = millis();
  if (now < next) {
    unsigned long elapsed = now - lastMeasurement;
    cum += (ina219_lines[currentLine].weighted_uA * (elapsed/5))/1000;
    lastMeasurement = now;
    return;
  }
  lineCalibs[currentLine].offsets[(currentDir ? 256 : 0) + currentPwm/LINECALIB_RATIO] = cum / ina219_count;

  if (currentLine > 3) {
    linecalib_calibrating = false;
    return true;
  }
  if (currentPwm > 255) {
    if (currentDir == false) {
      currentDir = true;
    } else {
      currentLine += LINECALIB_RATIO;
      currentDir = false;
    }
    currentPwm = 0;
    Serial.print("Z: next is line");
    Serial.print(currentLine + 'A');
    Serial.print(" dir");
    Serial.print(currentDir ? 'A' : 'B');
    Serial.print(" power");
    Serial.println(currentPwm);
  }
  Line *line;
  switch (currentLine) {
    case 0: line = &lineA; break;
    case 1: line = &lineB; break;
    case 2: line = &lineC; break;
    case 3: line = &lineD; break;
  }
  Line_setDirection(line, currentDir);
  Line_setPwm(line, currentPwm, false);

  next = millis() + 100;
  cum = 0;
  return false;
}
