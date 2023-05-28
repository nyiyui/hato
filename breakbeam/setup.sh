#!/usr/bin/env sh

echo 'NOTE add .cli-config.yml contents to your ~/.arduino*/arduino-cli.yml'
arduino-cli core update-index
arduino-cli core install adafruit:samd
