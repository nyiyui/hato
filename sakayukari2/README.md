# Sakayukari 2

- `var` is a mutable value, and can be used to implement reactive programming
- `val` is an immutable value
- `call` is a blocking call
- `pass` is for non-blocking calls

TODO: var and chaining passes seem similar for reactive programming
potential solution to TODO:
var is sugar for pass
```
(def SLCP (class
  (var id ID)
))
(def slcp (new SLCP))
(when :do {
  (print slcp.id)
})
```
↑ desugars to ↓
```
(def SLCP (class
  (pass id -> ID)
))
(def slcp (new SLCP))
(when (def id (<- slcp.id)) :do {
  (print id)
})
```

```
(def ID (data
  (val type String)
  (val variant String)
  (val instance String)
))
(def SLCP (class
  (var id ID)
  (call close () -> !io)
  (pass io <-> String)
))
```
