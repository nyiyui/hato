# 縁さか

## Goals

- several devices (try to) uphold relations
- relation
  - essentially an actor
  - can be aliased (e.g. if stateless)
  - can be stateful, side-effectful, etc
  - doing io brings guarantees around timing
  - can be propagated by:
    - changes in input (e.g. `C = (F-32)*5/9`)
    - "pulling" (仮) from its dependents (e.g. `current-timestamp` fetched from OS when requestsd)
    - side-effectful relations can be run arbitrarily (e.g. run 60 times a second, without overlap)

## Notes

- types of relations
  - lazy - used by eager relations (e.g. time now), can be triggered by anything
  - eager - recomputes on any change
  - clock - triggers a change every time interval
  - linear - runs do not overlap

```syk
actor io:
  f = parse int (readline)
c := solve {f-32*5/9} # eaher
actor io:
  forall:
    print (format {"{f}°F = {c}°C"})
```

```syk
line = ext "soyuu-line/-"
actor linear:
  states = new list of int
  selected: index of states = 0
  forall:
    match termui.key:
      "<Up>" => states[selected] ++
      "<Down>" => states[selected] --
      "<Left>" => selected --
      "<Right>" => selected ++ # ignore overflow and underflow for brevity
    each i, state of states:
      line[i] = state
```

## Provisional Syntax

actor "types":
- clock - triggered every e.g. 1/60 seconds
- lazy - triggered by a clock or other
- eager - "normal" actor - runs on changes, concurrent
- uniq - no conurrent runs (e.g. keypress actions) - propagates uniqueness
- io - stateful + linear

```syk
soyuu = import "soyuu";
velocity = import "velocity";

line = soyuu.conn "soyuu-line:leonardo/0";
breakbeam = velocity (soyuu.conn "soyuu-breakbeam:itsybitsy-m0/0");
inference = infer {
  sources = [
    { breakbeam; position = 12000 };
  ];
};
```
