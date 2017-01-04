# Building
*Building Ekanite requires Go 1.5 or later. [gvm](https://github.com/moovweb/gvm) is a great tool for installing and managing your versions of Go.*

Tested on 64-bit Kubuntu 14.04.

    mkdir ~/ekanite # Or a directory of your choice.
    cd ~/ekanite
    export GOPATH=$PWD
    go get github.com/ekanite/ekanite
    go install github.com/ekanite/...

# Cloning a fork
If you wish to work with a fork of Ekanite, your own fork for example, you must still follow the directory structure above. But instead of cloning the main repo, instead clone your fork. You must fork the project if you want to contribute upstream.

Follow the steps below to work with a fork:

```bash
    export GOPATH=$HOME/ekanite
    mkdir -p $GOPATH/src/github.com/ekanite
    cd $GOPATH/src/github.com/ekanite
    git clone git@github.com:<your Github username>/ekanite
```

Retaining the directory structure `$GOPATH/src/github.com/ekanite` is necessary so that Go imports work correctly.
