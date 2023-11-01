#!/usr/bin/env sh

arduino-cli core update-index
arduino-cli core install adafruit:samd
arduino-cli lib install "Adafruit Motor Shield V2 Library"
arduino-cli lib install "Adafruit INA219"
