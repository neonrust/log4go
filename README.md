# log4go #

Simple logging for Go akin to the well-known log4j.
The API was modeled after Python's [logging](https://docs.python.org/3/library/logging.html) module.

Most things are kept as simple as possible.

## Loggers ##

Hierarchies of loggers may be created. Slightly different from log4j and Python's logging module, the hierarchy is formatted like a file system path: `base/child/grandchild`. (log4j uses dots as separator; `base.child.grandchild`) Root logger has empty name.

Each `Logger` instance has at least one `Handler` associated to it.
The `Logger` might have a `Level` set, to limit the logging to that level and above. However, by default it has no level set, which means it will use the level from first ancestor that has it set.

## Handlers ##

Each handler has a formatter associated to it.

Included:

* `NewStreamHandler`
* `NewFileHandler`

## Formatters ##

* `NewTemplateFormatter`: Formats the message based on a template string. See below for syntax.



## Example ##

```
log4go.BasicConfig(log4go.BasicConfigOpts{
    Level: log4go.INFO,
    FileName: "awesome.log",
    Format: "{time} {name<10} {level<8} {message}",
})

// Using the root logger
rootLog := log4go.GetLogger()
rootLog.Debug("won't be shown")
rootLog.Info("Hello, log4go!")

// Using a sub-logger
// By top-level function:
myLog := log4go.GetLogger("mylog")
myLog.Error("No, not really")
// Or by another logger:
myLogAlso := rootLog.GetLogger("mylog")

myLog.Info("printf-formatting works, %s", "of course")
myLogAlso.Warning("this is dangerously awesome!")
```

The above will output:
```
2016-09-23 root      INFO     Hello, log4go!
2016-09-23 mylog     ERROR    No, not really
2016-09-23 mylog     WARNING  this is dangerously awesome!
```

## TemplateFormatter ##

Template syntax is similar to the string formatting language in Python (and possibly others).

Basic token syntax is: `{token}`
It might include an alignment and field width as well: `{token<width}`, e.g. `{name>20}` (right-aligned logger name).

Supported tokens are:

* name - Logger's full name.
* time - Time stamp in [ISO 8601](https://en.wikipedia.org/wiki/ISO_8601) format, but without time zone.
* level - Name of log message's level.
* message - The log message itself.
