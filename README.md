# log4go #

Simple logging for Go akin to the well-known log4j.
The API was modeled after Python's [logging](https://docs.python.org/3/library/logging.html) module.

Most things are kept as simple as possible. For example, the (currently) only way to configure the logging system is through code, most prominently via the `BasicConfig()` call. There is no file-based configuration.

## Loggers ##

Hierarchies of loggers may be created. The full logger name is formatted slightly different from log4j and Python's logging module; more akin to a file system path: `base/child/grandchild` (log4j uses dots as separator; e.g. `base.child.grandchild`). The root logger has no name (or rather, an empty string).

Any `Logger` instance may have any number of `Handler` associated to it. When a log record is issued, it starts at the `Logger` instance used and up towards the root, passing it to all `Handler` instances it finds along the way.
The `Logger` might have a `Level` set; dropping all messages that has a logging level below that level. However, by default it has no level set, which means it will use the level from the first ancestor that has it set. If none have level set, `WARNING` is the default.

## Handlers ##

A handler writes a log record the way it knows how, where/however that may be.

Included handlers:

* `NewStreamHandler`
* `NewFileHandler`

Each handler has a formatter associated to it (it's useless without it).

## Formatters ##

A formatter encodes a log record into a `[]byte` object (for the handler to then write).

Included formatters:

* `TemplateFormatter`: Formats the message based on a template string. See below for syntax.



## Example ##

```
:::go
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
2016-09-23 11:22:33 mylog/cool INFO     specific stuff
```

## TemplateFormatter ##

Template syntax is similar to the string formatting language in Python (and possibly others). Yes, I know, not Go templates... :(

Basic token syntax is: `{token}`
It might include an alignment and field width as well: `{token<width}`, e.g. `{name>20}` (right-aligned logger name).

Supported tokens are:

* `name` - Logger's full name.
* `time` - Time stamp in [ISO 8601](https://en.wikipedia.org/wiki/ISO_8601) format, but without time zone.
* `timems` - Same as `time`, but with milliseconds as well.
* `level` - Name of log message's level.
* `message` - The log message itself.