goagain
=======

(UNIX socket edition)

Zero-downtime restarts in Go
----------------------------

Inspired by [Unicorn](http://unicorn.bogomips.org/), the `goagain` package provides primitives for bringing zero-downtime restarts to Go applications that accept connections from a [`net.UnixListener`](http://golang.org/pkg/net/#UnixListener).

This is a derivative program from rcrowley's `https://github.com/rcrowley/goagain`.

Installation
------------

        git clone https://github.com/Rudd-O/goagain
        cd goagain
        export GOPATH=...
        go install goagain goagain-example

Usage
-----

[`goagain-example.go`](https://github.com/Rudd-O/goagain/blob/master/src/goagain-example/main.go) shows how it's done.  After building it, run it as:

        bin/goagain-example -socket=/tmp/goagain-example.sock

To connect you will need `socat` installed on your system:

        echo "GET /slow HTTP/1.0
        
        " | socat -t20 - UNIX-CONNECT:/tmp/goagain-example.sock

We've included a command line client in [`client/sget`](client/sget) that helps you perform HTTP requests against a UNIX socket.
