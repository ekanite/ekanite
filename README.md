# Ekanite [![Circle CI](https://circleci.com/gh/ekanite/ekanite/tree/master.svg?style=svg)](https://circleci.com/gh/ekanite/ekanite/tree/master) [![GoDoc](https://godoc.org/github.com/ekanite/ekanite?status.png)](https://godoc.org/github.com/ekanite/ekanite)
*Ekanite* is a syslog server with built-in search. Its goal is to do one thing, and do it well -- receive log messages over the network and allow those messages to be searched. What it lacks in feature, it makes up for in focus. Built in [Go](http://www.golang.org), it has no external dependencies, which makes deployment easy.

Features include:

- Full text search of all received log messages.
- Full parsing of [RFC5424](http://tools.ietf.org/html/rfc5424) headers.
- Log messages are indexed by parsed timestamp, if one is available. This means search results are presented in the order the messages occurred, not in the order they were received, ensuring sensible display even with delayed senders.
- Automatic data-retention management. Ekanite deletes indexed log data older than a configurable time period.

Building
------------
Tested on 64-bit Kubuntu 14.04.

    mkdir ~/ekanite # Or a directory of your choice.
    cd ~/ekanite
    export GOPATH=$PWD
    go get github.com/ekanite/ekanite
    go install github.com/ekanite/...

Running
------------
The daemon will be located in the ```$GOPATH/bin``` directory. Execute

    $ ekanited -h
        ekanite [options]
        -batchsize=300: Indexing batch size.
        -batchtime=1000: Indexing batch timeout, in milliseconds.
        -datadir="/var/opt/ekanite": Set data directory.
        -diag="": expvar and pprof bind address in the form host:port. If not set, not started.
        -maxpending=1000: Maximum pending index events.
        -noreport=false: Do not report anonymous data on launch.
        -numshards=16: Set number of shards per index.
        -query="localhost:9950": TCP Bind address for query server in the form host:port.
        -retention="168h": Data retention period. Minimum is 24 hours.
        -tcp="": Syslog server TCP bind address in the form host:port. If not set, not started.
        -udp="": Syslog server UDP bind address in the form host:port. If not set, not started.

for command-line options.

Sending logs to Ekanite
------------
For now, for ekanite to accept logs, your syslog client must be configured such that the log lines are [RFC5424](http://tools.ietf.org/html/rfc5424) compliant, and in the following format:

    <PRI>VERSION TIMESTAMP HOSTNAME APP-NAME PROC-ID MSGID MSG"

Consult the RFC to learn what each of these fields is. The TIMESTAMP field must be in [RFC3339](http://www.ietf.org/rfc/rfc3339.txt) format.  Both [rsyslog](http://www.rsyslog.com/) and [syslog-ng](http://www.balabit.com/network-security/syslog-ng) support templating, which make it easy to format messages correctly. For example, an rsyslog template looks like so:

    $template Ekanite,"<%pri%>%protocol-version% %timestamp:::date-rfc3339% %HOSTNAME% %app-name% %procid% - %msg%"

syslog-ng looks like so:

    template Ekanite { template("<${PRI}>1 ${ISODATE} ${HOST} ${PROGRAM} ${PID} - $MSG"); template_escape(no) };
    
Searching the logs
------------
Search support is pretty simple at the moment. Telnet to the query server (see the command line options) and enter a search term. The query language supported is the simple language supported by [bleve](http://godoc.org/github.com/blevesearch/bleve#NewQueryStringQuery), but a more sophisiticated query syntax, including searching for specific field values, will be supported soon.

