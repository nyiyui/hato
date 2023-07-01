#!/usr/bin/env sh

set -eux

port='/dev/ttyACM0'
fqbn='adafruit:samd:adafruit_feather_m4'
arduino-cli compile --fqbn "$fqbn"
arduino-cli upload -p "$port" --fqbn "$fqbn"
#sleep 2
#picocom "$port"