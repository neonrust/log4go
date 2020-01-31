# log4go #

Simple logging for Go akin to the well-known log4j.  The API was
modeled after Python's
[logging](https://docs.python.org/3/library/logging.html) module.

Most things are kept as simple as possible. For example, the
(currently) only way to configure the logging system is through code,
most prominently via the `BasicConfig()` call. There is no file-based
configuration.

## Dependency-free ##

Completly free of external dependencies.
Only Go's standard library is used.


## Loggers ##

Hierarchies of loggers may be created, just by calling `GetLogger()`
on any logger to add a child. There's no way to remove loggers at the
moment. Not a problem to implement, a need just never arised. :)

Any `Logger` instance may have any number of `Handler` instances
associated to it. When a log message is issued, it starts at the
`Logger` instance invoked on and up towards the root, passing the
message to all `Handler` instances it finds along the way.

Each `Handler` has a `Formatter` associated to it (it's useless
without it). A default one if no

The `Logger` might have a `Level` set; dropping all messages that has
a logging level below that level. However, by default it has no level
set, which means it will use the level from the first ancestor with a
set level. If none of the ancestors have level set, `WARNING` will be
used. The level check is performed in the calling goroutine
as-soon-as-possible, e.g. before any message formatting.

The full logger name (used in the log file) is
formatted slightly different from log4j and Python's logging module;
more akin to a file system path: `base/child/grandchild` (log4j uses
dots as separator; e.g. `base.child.grandchild`). The root logger has
no name (or rather, an empty string).


## Handlers ##

A handler writes a log message the way it knows how, where/however that may be.

Included handlers:

* `StreamHandler`
* `FileHandler`
* `WatchedFileHandler`


A slightly more detailed description of these are at the bottom.


## Formatters ##

A `Formatter` encodes a log message into a `[]byte` object. This is
then used by the `Handler` to write it.

Included formatters:

* `TemplateFormatter`: Formats the message based on a template
  string. See below for syntax of this string.


## TemplateFormatter ##

Template syntax is similar to the string formatting language in Python
(and others). Yes, I know, not Go templates... :(

Basic token syntax is: `{token}`.

Width and alignment can also be specified using a slightly expanded
syntax: `{token#width}` Where `#` is either `<` (meaning left-aligned)
or `>` (right-aligned). If the string exceeds the specified width, it
wont be truncated, but alignment will of course be useless. This
behaviour should not be surprising to anyone...

Supported tokens are:

* `name` - Logger's full name.
* `basename` - Logger's name (last part).
* `time` - Time stamp in [RFC 3339](https://tools.ietf.org/html/rfc3339) format, but without time zone, and no `T`.
* `timems` - Same as `time`, but with milliseconds as well.
* `level` - Name of log message's level.
* `message` - The log message text.

No, there's no way to control how the time is formatted. I'm using the one, true format.


## Example ##

```go
log4go.BasicConfig(log4go.BasicConfigOpts{
    Level: log4go.INFO,
    FileName: "awesome.log",
    Format: "{time} {name<10} {level<8} {message}",
})

// Since commit 21eaadd, StreamHandler now writes to the stream in a goroutine.
// To make sure it's flushed at app exit, call Shutdown().
defer log4go.Shutdown()

// Using the root logger
rootLog := log4go.GetLogger()
rootLog.Debug("won't be shown")
rootLog.Info("Hello, log4go!")

// Using a specific logger
// By top-level function:
myLog := log4go.GetLogger("mylog")
myLog.Error("Awesomeness ahead")
// Or by another logger:
myLogAlso := rootLog.GetLogger("mylog")

myLog.Info("printf-formatting works, %s", "of course")
myLogAlso.Warning("dangerously useful")

// Using a sub-logger (inherits parent's log level, unless further restricted)
subLog := myLog.GetLogger("cool")
subLog.Info("specific stuff")
```

The above will output:
```
2016-09-23 11:22:33 root       INFO     Hello, log4go!
2016-09-23 11:22:33 mylog      ERROR    Awesomeness ahead
2016-09-23 11:22:33 mylog      WARNING  dangerously useful
2016-09-23 11:22:33 mylog      INFO     printf-formatting works, of course
2016-09-23 11:22:33 mylog/cool INFO     specific stuff
```

## Included Handlers ##

* `StreamHandler`

Writes messages to an `io.Writer`.  This should arguably
be renamed `WriterHandler` to align better with Go interface names.

* `FileHandler`

This inherits from `StreamHandler`. It opens the specified file,
optionally appending, than passes it on to `StreamHandler`.

* `WatchedFileHandler`

This wraps a `StreamHandler`. It adds a check _at each message_
whether the destination file has moved (an indication that log
rotation has occurred), and if so re-opens the file.

This might be a sane default handler, if it wasn't for the
performance-hit the check brings. However, if you're not
super-critical of logging performance, this is fine to use. No
benchmarks have been performed though. ;)
