#include <Wire.h>
#include <SPI.h>
#include <Adafruit_PN532.h>
#include <Adafruit_NeoPixel.h>

#define VARIANT "adafruit:samd:adafruit_feather_m4/0"

struct meta {
  unsigned long pos; // position of the RFID sensor coil
};

static struct meta meta = {
  .pos = 0,
};

// built-in NeoPixel (Feather M4)
Adafruit_NeoPixel strip(1, 8, NEO_GRB + NEO_KHZ800);

/** Based on sample code (readMifare.pde) generously provided by Adafruit
 * Software License Agreement (BSD License)
 * 
 * Copyright (c) 2012, Adafruit Industries
 * All rights reserved.
 * 
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 * 1. Redistributions of source code must retain the above copyright
 * notice, this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright
 * notice, this list of conditions and the following disclaimer in the
 * documentation and/or other materials provided with the distribution.
 * 3. Neither the name of the copyright holders nor the
 * names of its contributors may be used to endorse or promote products
 * derived from this software without specific prior written permission.
 * 
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS ''AS IS'' AND ANY
 * EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 * WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 * DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER BE LIABLE FOR ANY
 * DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 * (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 * LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 * ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 * SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

// If using the breakout with SPI, define the pins for SPI communication.
#define PN532_SCK  (25)
#define PN532_MOSI (24)
#define PN532_SS   (13)
#define PN532_MISO (23)

// If using the breakout or shield with I2C, define just the pins connected
// to the IRQ and reset lines.  Use the values below (2, 3) for the shield!
//#define PN532_IRQ   (2)
//#define PN532_RESET (3)  // Not connected by default on the NFC Shield

// Uncomment just _one_ line below depending on how your breakout or shield
// is connected to the Arduino:

// Use this line for a breakout with a software SPI connection (recommended):
//Adafruit_PN532 nfc(PN532_SCK, PN532_MISO, PN532_MOSI, PN532_SS);

// Use this line for a breakout with a hardware SPI connection.  Note that
// the PN532 SCK, MOSI, and MISO pins need to be connected to the Arduino's
// hardware SPI SCK, MOSI, and MISO pins.  On an Arduino Uno these are
// SCK = 13, MOSI = 11, MISO = 12.  The SS line can be any digital IO pin.
Adafruit_PN532 nfc(PN532_SS);

// Or use this line for a breakout or shield with an I2C connection:
//Adafruit_PN532 nfc(PN532_IRQ, PN532_RESET);

// Or use hardware Serial:
//Adafruit_PN532 nfc(PN532_RESET, &Serial1);

void setup() {
  strip.begin();
  strip.show();
  strip.setPixelColor(0, 32, 32, 0);
  strip.show();
  Serial.begin(115200);
  while (!Serial) delay(10); // the whole point of this board is to transmit RFID data using serial...

  Serial.println("init: PN53x");
  nfc.begin();

  uint32_t versiondata = nfc.getFirmwareVersion();
  if (! versiondata) {
    strip.setPixelColor(0, 255, 0, 0);
    strip.show();
    Serial.println("Didn't find PN53x board");
    Serial.println(" Serror");
    while (1); // halt
  }
  Serial.print("Found chip PN5"); Serial.println((versiondata>>24) & 0xFF, HEX);
  Serial.print("Firmware ver. "); Serial.print((versiondata>>16) & 0xFF, DEC);
  Serial.print('.'); Serial.println((versiondata>>8) & 0xFF, DEC);

  Serial.println(" Sready");
  strip.setPixelColor(0, 32, 32, 32);
  strip.show();
}

void loop() {
  handleSLCP();
  readRFID();
}

void handleSLCP() {
  static char buffer[11];
  if (Serial.available() == 0)
    return;
  int kind = Serial.read();
  if (kind == 'I') {
    Serial.println(" Isoyuu-rfid/" VARIANT);
    int eol = Serial.read();
    if (eol != '\n') {
      Serial.println(" Eexpected eol");
    }
  } else if (kind == 'J') {
    int eol = Serial.read();
    if (eol != '\n') {
      Serial.println(" Eexpected eol");
    }
    Serial.print(" Isoyuu-rfid/" VARIANT ";");
    Serial.print("J0P");
    Serial.print(meta.pos);
    Serial.println();
  } else {
    Serial.print(" Eunknown kind ");
    Serial.println(kind);
  }
}

void readRFID() {
  uint8_t success;
  uint8_t uid[] = { 0, 0, 0, 0, 0, 0, 0 };  // Buffer to store the returned UID
  uint8_t uidLength;                        // Length of the UID (4 or 7 bytes depending on ISO14443A card type)

  // Wait for an ISO14443A type cards (Mifare, etc.).  When one is found
  // 'uid' will be populated with the UID, and uidLength will indicate
  // if the uid is 4 bytes (Mifare Classic) or 7 bytes (Mifare Ultralight)
  success = nfc.readPassiveTargetID(PN532_MIFARE_ISO14443A, uid, &uidLength);

  if (success) {
    Serial.print(" Dcard1 L");
    Serial.print(uidLength, DEC);
    Serial.print(" V");
    for (int i = 0; i < 7; i ++)
      Serial.print(uid[i], HEX);
    Serial.println();
    // Serial.println("Found an ISO14443A card");
    // Serial.print("  UID Length: ");Serial.print(uidLength, DEC);Serial.println(" bytes");
    // Serial.print("  UID Value: ");
    // nfc.PrintHex(uid, uidLength);
    // if (uidLength == 4)
    // {
    //   // We probably have a Mifare Classic card ...
    //   uint32_t cardid = uid[0];
    //   cardid <<= 8;
    //   cardid |= uid[1];
    //   cardid <<= 8;
    //   cardid |= uid[2];
    //   cardid <<= 8;
    //   cardid |= uid[3];
    //   Serial.print("Seems to be a Mifare Classic card #");
    //   Serial.println(cardid);
    // }
    // Serial.println("");
  }
}

