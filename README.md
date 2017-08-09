# GelfFormatter

This is a [logrus](https://github.com/sirupsen/logrus) formatter which formats log messages into valid [gelf](http://docs.graylog.org/en/2.2/pages/gelf.html) for [graylog](https://www.graylog.org/)

This library is hugely based on the [graylog gelf library](https://github.com/Graylog2/go-gelf) but differs in that it is complaint with [12 factor log principles](https://12factor.net/logs), which states ... 

```text
A twelve-factor app never concerns itself with routing or storage of its output stream.
It should not attempt to write to or manage logfiles.
Instead, each running process writes its event stream, unbuffered, to stdout.
During local development, the developer will view this stream in the foreground of their terminal to observe the app’s behavior.

In staging or production deploys, each process’ stream will be captured by the execution environment, collated together with all other streams from the app, and routed to one or more final destinations for viewing and long-term archival.
These archival destinations are not visible to or configurable by the app, and instead are completely managed by the execution environment.
Open-source log routers (such as Logplex and Fluent) are available for this purpose.
```

### Installation

```bash
go get -u github.com/ballad89/gelfFormatter
```

```go
package main

import (
    "github.com/ballad89/gelfFormatter"
    log "github.com/sirupsen/logrus"
    
)

func main() {

    f, err := gelfFormatter.NewGelfFormatter("application-name")

    if err != nil {
        panic(err)
    }
    log.SetFormatter(f)

    log.SetOutput(os.Stdout)

    log.WithFields(log.Fields{
        "animal": "walrus",
    }).Info("animal")
}
```


```text
{"version":"1.1","host":"machine.name","short_message":"animal","timestamp":1502245262,"level":6,"_facility":"application-name","_animal":"walrus"}
```
