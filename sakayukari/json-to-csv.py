#!/usr/bin/env python3

import sys, json, csv

print('waiting for JSON from stdin...', file=sys.stderr)
data = json.load(sys.stdin)
w = csv.writer(sys.stdout)
w.writerow(['power', 'speed'])
w.writerows(data['Points'])
