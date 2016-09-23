# log4go #

Simple logging for Go akin to the well-known log4j.
The API was modeled after Python's [logging](https://docs.python.org/3/library/logging.html) module.

```
log4go.BasicConfig(log4go.BasicConfigOpts{
    Level: log4go.INFO,
    FileName: "awesome.log",
})

rootLogger := log4go.GetLogger()
rootLogger.Debug("won't be shown")
rootLogger.Info("Hello, log4go!")
```

Will output:
```
Hello, log4go!
```