# Yosuka

```
(print "prints this string to *somewhere*")
(print 123) # can do anything (just prints their representation
```

```
(if cond then a else b)
```

`cond` is always executed, and depdning on the value, `a` or `b` is executed.

```
(def qa (quote a)) # a is not executed
(exec qa) # a is finally executed
```

- where normally effects are used, actors and their corresponding messages are used
  - what about overriding or having multiple handlers
- where normally continuations are used, an actor just waits for a message and the schedulaer runs other actors

```yos
(def UART (class
  (def Control (type (or
    Close
    Change
  )))
  (ch linear data <-> String!io)
  (call control Control -> !io)
))
```

```
(exec (print "abc") (print "def")) # executes all arguments in order
(print "for debugging") # prints to *somewhere* for debugging
```

# OLD VERSION BELOW

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
