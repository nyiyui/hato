#include <Adafruit_MotorShield.h>
#include "ina219.h"

typedef struct Line {
    Adafruit_DCMotor *motor;
    bool direction;
} line;

Adafruit_MotorShield shield0 = Adafruit_MotorShield();

static Line lineA = { .motor = shield0.getMotor(1), .direction = true, };
static Line lineB = { .motor = shield0.getMotor(2), .direction = true, };
static Line lineC = { .motor = shield0.getMotor(3), .direction = true, };
static Line lineD = { .motor = shield0.getMotor(4), .direction = true, };

void Line_setDirection(Line *line, bool direction) {
    line->direction = direction;
}

void Line_setPwm(Line *line, int value, bool brake) {
    line->motor->setSpeed(value);
    if (!value) {
        line->motor->run(RELEASE);
        return;
    }
    line->motor->run(brake ? BRAKE : (line->direction ? FORWARD : BACKWARD));
}

void setup() {
    Serial.println(" Sstart");
    // === Motor Shield
    while (!shield0.begin()) {
        Serial.println(" Smotor shield init failed");
        delay(1000);
    }
    // === INA219
    ina219_init();
    Serial.println(" Swait 5 seconds...");
    delay(5000);
    Serial.println(" Sready");
}

// #define DEFINE_GET_CURRENT(ina219_i, ina219) \
//     long getCurrent_ ## ina219_i() { \
//         static long buffer[8] = {0}; \
//         static int bufferIndex = 0; \
//         buffer[bufferIndex] = ina219.getCurrent_mA() * 1000.0; \
//         bufferIndex ++; \
//         bufferIndex % 8; \
//         long sum = 0; \
//         for (int i = 0; i < 8; i ++) \
//             sum += buffer[i]; \
//         return sum / 8; \
//     }
// 
// DEFINE_GET_CURRENT(0, ina2190);
// DEFINE_GET_CURRENT(1, ina2191);
// DEFINE_GET_CURRENT(2, ina2192);
// DEFINE_GET_CURRENT(3, ina2193);

void loop() {
    static unsigned long next = 0;
    unsigned long now = millis();
    if (next <= now) {
        ina219_update();
        next = now + 5;
        Serial.print("weighted0_uA:");
        Serial.println(ina219_lines[0].weighted_uA);
    }
    handleSLCP();
    // long c = getCurrent_0();
    // Serial.print("current0: ");
    // Serial.println(c);
    // TODO: getCurrent hangs serial - why?
    // Serial.print("currentRaw: ");
    // Serial.println(ina2190.getCurrent_mA());
}

void handleShort(bool isShort) {
    static char digitBuffer[4];
    Serial.println(" Psc");
    // TODO: error checking
    // Serial.readBytesUntil('\n', buffer, 10);
    int line = Serial.read();
    Serial.print(" Pline ");
    Serial.println(line);
    Line *t;
    if (line == 'A') t = &lineA;
    else if (line == 'B') t = &lineB;
    else if (line == 'C') t = &lineC;
    else if (line == 'D') t = &lineD;
    else {
        Serial.print(" Einvalid line ");
        Serial.println(line);
        return;
    }
    int direction = Serial.read();
    int brake = Serial.read();
    int read = Serial.readBytes(digitBuffer, 3);
    if (read != 3) {
        Serial.println(" Enot enough");
        return;
    }
    int speed = atoi(digitBuffer);
    if (speed < 0 || speed > 255) {
        Serial.println(" Eout of range");
        return;
    };
    int eol = Serial.read();
    if (eol != '\n') {
        Serial.println(" Eexpected eol");
        return;
    }
    Serial.print(" Pact dir");
    Serial.print(direction);
    Serial.print(" brk");
    Serial.print(brake);
    Serial.print(" spd");
    Serial.print(speed);
    Serial.println(".");

    Serial.println(" Pstarting");
    if (direction == 'A') Line_setDirection(t, true);
    else if (direction == 'B') Line_setDirection(t, false);
    Line_setPwm(t, speed, brake == 'Y');
    if (isShort) {
        Serial.println(" Pwaiting");
        delay(100);
        Serial.println(" Pstopping");
        Line_setPwm(t, 0, true);
    }
    Serial.println(" Ook");
    return;
}

// handle in/out of soyuu line control protocol
void handleSLCP() {
    static char buffer[11];
    if (Serial.available() == 0) return;
    int kind = Serial.read();
    if (kind == 'I') {
        Serial.println(" Isoyuu-line-mega-0");
        int eol = Serial.read();
        if (eol != '\n') {
            Serial.println(" Eexpected eol");
        }
    } else if (kind == 'S') {
        handleShort(true);
    } else if (kind == 'C') {
        handleShort(false);
    } else if (kind == 'D') {
        buffer[0] = Serial.read();
        buffer[1] = Serial.read();
        buffer[2] = Serial.read();
        buffer[3] = '\0';
        ina219_weight = atoi(buffer);
        Serial.print("ina219_weight set to ");
        Serial.print(ina219_weight);
        Serial.println(". Note: this is only saved to RAM.");
    } else {
        Serial.print(" Eunknown kind ");
        Serial.println(kind);
    }
}
