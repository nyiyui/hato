#!/usr/bin/python3

import json
import sys

res = {}

for line in sys.stdin:
    data = json.loads(line)
    res[data['Power']] = data["Attitude"]["Velocity"]

json.dump(res, sys.stdout)
