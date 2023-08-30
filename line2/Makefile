port=/dev/ttyACM0
fqbn=adafruit:samd:adafruit_itsybitsy_m0

upload:
	arduino-cli compile --fqbn "${fqbn}"
	arduino-cli upload -p "${port}" --fqbn "${fqbn}"

setup:
	echo 'add .cli-config.yml contents to your ~/.arduino*/arduino-cli.yml'
	arduino-cli core install adafruit:samd
	arduino-cli lib install 

.PHONY: upload setup
