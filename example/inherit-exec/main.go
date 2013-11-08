package main

import (
	"github.com/rcrowley/goagain"
	"log"
	"net"
	"sync"
	"time"
)

func init() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
}

func main() {
	var (
		err error
		l net.Listener
		ppid int
	)

	// Get the listener and ppid from the environment.  If this is successful,
	// this process is a child that's inheriting and open listener from ppid.
	l, ppid, err = goagain.GetEnvs()

	ch := make(chan struct{})
	wg := &sync.WaitGroup{}
	if nil != err {

		// Listen on a TCP or a UNIX domain socket (the latter is commented).
		laddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:48879")
		if nil != err {
			log.Fatalln(err)
		}
		log.Printf("listening on %v", laddr)
		l, err = net.ListenTCP("tcp", laddr)
		/*
			laddr, err := net.ResolveUnixAddr("unix", "127.0.0.1:48879")
			if nil != err {
				log.Fatalln(err)
			}
			log.Printf("listening on %v", laddr)
			l, err = net.ListenUnix("unix", laddr)
		*/
		if nil != err {
			log.Fatalln(err)
		}

		// Accept connections in a new goroutine.
		go serve(l, ch, wg)

	} else {

		// Resume listening and accepting connections in a new goroutine.
		log.Printf("resuming listening on %v", l.Addr())
		go serve(l, ch, wg)

		// Kill the parent, now that the child has started successfully.
		if err := goagain.KillParent(ppid); nil != err {
			log.Fatalln(err)
		}

	}



/*

inherit parent:

1. AwaitSignals (SIGUSR2)
2. Relaunch (rename this, please; it's not *really* public API, anyway)
3. AwaitSignals (SIGQUIT)

inherit child:

1. GetEnvs
2. go serve
3. KillParent (SIGQUIT)
4. AwaitSignals

----

inherit-exec parent:

1. AwaitSignals (SIGUSR2)
2. Relaunch (rename this, please; it's not *really* public API, anyway)
3. AwaitSignals (SIGUSR2)
4. Exec (TODO name)
5. KillChild (SIGQUIT)

inherit-exec child:

1. GetEnvs
2. go serve
3. KillParent (SIGUSR2)
4. AwaitSignals

*/




	// Block the main goroutine awaiting signals.
	if err := goagain.AwaitSignals(l); nil != err {
		log.Fatalln(err)
	}

	// Do whatever's necessary to ensure a graceful exit like waiting for
	// goroutines to terminate or a channel to become closed.
	//
	// In this case, we'll simply stop listening and wait one second.
	if err := l.Close(); nil != err {
		log.Fatalln(err)
	}
	time.Sleep(1e9)

	// Now re-exec the parent process

}

func serve(l net.Listener, ch chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ch:
			break
		default:
		}
		l.SetDeadline(time.Now().Add(100e6))
		c, err := l.Accept()
		if nil != err {
			if goagain.IsErrClosing(err) || err.(*net.OpError).Timeout() {
				break
			}
			log.Fatalln(err)
		}
		c.Write([]byte("Hello, world!\n"))
		c.Close()
	}
}
