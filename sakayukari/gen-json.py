#!/usr/bin/env python3
import sys
import json

power = []
speed = []
target = None
for line in sys.stdin:
    if line.strip() == "power":
        target = power
    elif line.strip() == "speed":
        target = speed
    else:
        if target == power:
            target.append(int(line.strip()))
        elif target == speed:
            target.append(round(float(line.strip())))

print(f"power: {power}", file=sys.stderr)
print(f"speed: {speed}", file=sys.stderr)
if len(power) != len(speed):
    raise TypeError(f"power and speed lengths do not match")

data = dict(Points=[])
for i in range(len(power)):
    data['Points'].append([power[i], speed[i]])

json.dump(data, sys.stdout)
