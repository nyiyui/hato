#!/usr/bin/env bash
set -eux

arduino-cli compile --fqbn arduino:avr:mega
arduino-cli upload -p /dev/ttyACM0 --fqbn arduino:avr:mega
