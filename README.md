# Ekanite [![Circle CI](https://circleci.com/gh/ekanite/ekanite/tree/master.svg?style=svg)](https://circleci.com/gh/ekanite/ekanite/tree/master)
*Ekanite* is a syslog server with built-in search.

Building
------------
Tested on 64-bit Kubuntu 14.04.

    mkdir ~/ekanite # Or a directory of your choice.
    cd ~/ekanite
    export GOPATH=$PWD
    go get github.com/ekanite/ekanite
    go install github.com/ekanite/ekanite

Running
------------
The daemon will be located in the ```$GOPATH/bin``` directory. Execute

        ekanited -h

for command-line options.
