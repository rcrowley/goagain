package main

import (
	"github.com/rcrowley/goagain"
	"fmt"
	"log"
	"net"
	"sync"
	"syscall"
	"time"
)

func init() {
	goagain.Strategy = goagain.InheritExec
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	log.SetPrefix(fmt.Sprintf("pid:%d ", syscall.Getpid()))
}

func main() {

	// Inherit a net.Listener from our parent process or listen anew.
	ch := make(chan struct{})
	wg := &sync.WaitGroup{}
	wg.Add(1)
	l, err := goagain.Listener()
	if nil != err {

		// Listen on a TCP or a UNIX domain socket (TCP here).
		l, err = net.Listen("tcp", "127.0.0.1:48879")
		if nil != err {
			log.Fatalln(err)
		}
		log.Printf("listening on %v", l.Addr())

		// Accept connections in a new goroutine.
		go serve(l, ch, wg)

	} else {

		// Resume listening and accepting connections in a new goroutine.
		log.Printf("resuming listening on %v", l.Addr())
		go serve(l, ch, wg)

		// If this is the child, send the parent SIGUSR2.  If this is the
		// parent, send the child SIGQUIT.
		if err := goagain.Kill(); nil != err {
			log.Fatalln(err)
		}

	}



/*

inherit parent:

1. Wait (SIGUSR2)
2. ForkExec
3. Wait (SIGQUIT)
4. ...

inherit child:

1. Listner
2. go serve
3. Kill (SIGQUIT)
4. Wait (we're now the parent and back at the beginning)

----

inherit-exec parent:

1. Wait (SIGUSR2)
2. ForkExec
3. Wait (SIGUSR2)
4. ...
5. Exec
6. Kill (SIGQUIT)
7. Wait (we're still the parent and back at the beginning)

inherit-exec child:

GOAGAIN_FD
GOAGAIN_NAME
GOAGAIN_PID
GOAGAIN_SIGNAL

1. Listener
2. go serve
3. Kill (SIGUSR2)
4. Wait (SIGQUIT)
5. ...

*/




	// Block the main goroutine awaiting signals.
	sig, err := goagain.Wait(l)
	if nil != err {
		log.Fatalln(err)
	}

	// Do whatever's necessary to ensure a graceful exit like waiting for
	// goroutines to terminate or a channel to become closed.
	//
	// In this case, we'll close the channel to signal the goroutine to stop
	// accepting connections and wait for the goroutine to exit.
	close(ch)
	wg.Wait()

	// Now re-exec the parent process.
	if goagain.SIGUSR2 == sig {
		if err := goagain.Exec(l); nil != err {
			log.Fatalln(err)
		}
	}

}

func serve(l net.Listener, ch chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ch:
			break
		default:
		}
		l.(*net.TCPListener).SetDeadline(time.Now().Add(100e6)) // XXX
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
