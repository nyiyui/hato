# Rikien (力緑)

Reactive programming language to implement HATC/O (hopefully automatic train control/operation).

```
   stA
   ↓
A----------B
      \----C
      ↑
      mA
```

```rikien
connA = conn {
  type = "soyuu-dist"
  variant = null
}
lA = line { conn = conn connA "A" }
lB = line { conn = conn connA "B" }
lC = line { conn = conn connA "C" }
mA = lineMux {
  conn = conn connA "D"
  states = {
    (lA  lB) = {}
    (lA  lC) = {}
  }
}
stA = sst {
  conn = conn connB "A"
  kind = pointIR
}
predictor = predictor { ... }

# defines a new actor that changes mA.lock
[
  mA.lock = predictor.query {
    line = lA | lB
    pos = mA.pos.radius (20 mm)
  }
]
```
