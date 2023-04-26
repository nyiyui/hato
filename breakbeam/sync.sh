#!/usr/bin/env sh

set -eux

fqbn='adafruit:samd:adafruit_itsybitsy_m0'
arduino-cli compile --fqbn "$fqbn"
arduino-cli upload -p /dev/ttyACM0 --fqbn "$fqbn"
sleep 2
picocom /dev/ttyACM0
