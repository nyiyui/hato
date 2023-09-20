#!/usr/bin/env sh

set -eux

port='/dev/ttyACM0'
fqbn='arduino:avr:leonardo'
arduino-cli compile --fqbn "$fqbn"
arduino-cli upload -p "$port" --fqbn "$fqbn"
