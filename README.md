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

```
# Send messages to Ekanite over TCP using the template. Assumes Ekanite is listening on 127.0.0.1:5514
$template Ekanite,"<%pri%>%protocol-version% %timestamp:::date-rfc3339% %HOSTNAME% %app-name% %procid% - %msg%"
*.*             @@127.0.0.1:5514;EkaniteFormat
```

syslog-ng looks like so:

```
template Ekanite { template("<${PRI}>1 ${ISODATE} ${HOST} ${PROGRAM} ${PID} - $MSG"); template_escape(no) };
```

Searching the logs
------------
Search support is pretty simple at the moment. Telnet to the query server (see the command line options) and enter a search term. The query language supported is the simple language supported by [bleve](http://godoc.org/github.com/blevesearch/bleve#NewQueryStringQuery), but a more sophisiticated query syntax, including searching for specific field values, will be supported soon.

For example, below is an example search session, showing accesses to the login URL of a Wordpress site. The telnet clients connects to the query server and enters the string `login`

```
$ telnet 127.0.0.1 9950
Trying 127.0.0.1...
Connected to 127.0.0.1.
Escape character is '^]'.
login
<134>0 2015-05-06T01:24:41.232890+00:00 fisher apache-access - - 104.140.83.221 - - [06/May/2015:01:24:40 +0000] "GET /wp-login.php?action=register HTTP/1.0" 200 206 "http://www.philipotoole.com/" "Opera/9.80 (Windows NT 6.2; Win64; x64) Presto/2.12.388 Version/12.17"
<134>0 2015-05-06T01:24:41.232895+00:00 fisher apache-access - - 104.140.83.221 - - [06/May/2015:01:24:40 +0000] "GET /wp-login.php?action=register HTTP/1.1" 200 243 "http://www.philipotoole.com/wp-login.php?action=register" "Opera/9.80 (Windows NT 6.2; Win64; x64) Presto/2.12.388 Version/12.17"
<134>0 2015-05-06T02:47:54.612953+00:00 fisher apache-access - - 184.68.20.22 - - [06/May/2015:02:47:51 +0000] "GET /wp-login.php HTTP/1.1" 200 243 "-" "Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.1 (KHTML, like Gecko) Chrome/24.0.1309.0 Safari/537.17"
<134>0 2015-05-06T04:20:49.008609+00:00 fisher apache-access - - 193.104.41.186 - - [06/May/2015:04:20:46 +0000] "POST /wp-login.php HTTP/1.1" 200 206 "-" "Opera 10.00"
<134>0 2015-05-05T23:50:17.025568+00:00 fisher apache-access - - 65.98.59.154 - - [05/May/2015:23:50:12 +0000] "GET /wp-login.php HTTP/1.0" 200 206 "-" "-"
```

Perhaps you only want `POST` accesses to that URL:

```
login -GET
<134>0 2015-05-06T04:20:49.008609+00:00 fisher apache-access - - 193.104.41.186 - - [06/May/2015:04:20:46 +0000] "POST /wp-login.php HTTP/1.1" 200 206 "-" "Opera 10.00"
```

## Diagnostics
If diagnostics are enabled via the `-diag` switch, basic statistics and diagnostics will be available at the specified `host:port`. Simply visit `http://host:port/debug/vars` to retrieve this information.

## Reporting
Ekanite reports a small amount anonymous data to [Loggly](http://www.loggly.com), each time it is launched. This data is just the host operating system and system architecture and is only used to track the number of Ekanite deployments. Reporting can be disabled by passing `-noreport=true` to Ekanite at launch time.

