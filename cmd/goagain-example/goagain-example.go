package main

import (
	"github.com/rcrowley/goagain"
	"log"
	"net"
	"os"
)

func main() {

	// Get the listener and ppid from the environment.  If this is successful,
	// this process is a child that's inheriting and open listener from ppid.
	l, ppid, err := goagain.GetEnvs()

	if nil != err {

		// Listen on a TCP socket and accept connections in a new goroutine.
		laddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:48879")
		if nil != err {
			log.Println(err)
			os.Exit(1)
		}
		log.Printf("listening on %v", laddr)
		l, err = net.ListenTCP("tcp", laddr)
		if nil != err {
			log.Println(err)
			os.Exit(1)
		}
		go serve(l)

	} else {

		// Resume listening and accepting connections in a new goroutine.
		log.Printf("resuming listening on %v", l.Addr())
		go serve(l)

		// Kill the parent, now that the child has started successfully.
		if err := goagain.KillParent(ppid); nil != err {
			log.Println(err)
			os.Exit(1)
		}

	}

	// Block the main goroutine awaiting signals.
	if err := goagain.AwaitSignals(l); nil != err {
		log.Println(err)
		os.Exit(1)
	}

	// Do whatever's necessary to ensure a graceful exit like waiting for
	// goroutines to terminate or a channel to become closed.

}

func serve(l *net.TCPListener) {
	log.Println("TODO l.Accept()")
}
