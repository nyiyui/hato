# 緑阪 (Sakayukari)

Reactive language for model train control.

Effects:
- `ndet` - nondeterministic
- `state` - stateful
- `io` - guarantees actor invocation order, implies `ndet` and `state`

```ysk
ir-breakbeam := actor with: [ io ] body: {
  # no state relying on graph state, so no `state` marker
  dev := conn-dev id: "soyuu-breakbeam-0"
  loop {
    yield (dev recv)
  }
}
train-attitudes := actor with: [ state ] body: {
  some-state := state new
  attitudes := map new
  # set attitudes
  attitudes # return value
}
crossing := actor with: [] body: {
  # checks whether trains are approaching crossing
}
crossing-gate := actor with: [ io ] body: {
  dev := conn-dev id: "soyuu-crossingGate-0"
  crossing if-true: {
    dev send: "Cgl\n" # lower gate
  } else: {
    dev send: "Cgr\n" # raise gate
  }
}
```
