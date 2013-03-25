// Zero-downtime restarts in Go.
package goagain

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
        "runtime"
        "time"
        "sync"
        "net/http"
)

type ReqCounter struct {
        m sync.Mutex
        c int
}

func (c ReqCounter)get() (ct int) {
        c.m.Lock()
        ct = c.c
        c.m.Unlock()
        return
}

var reqCount ReqCounter

type SupervisedConn struct {
        net.Conn
}

func (w SupervisedConn) Close() error {
        log.Printf("close on conn to %v", w.RemoteAddr())
        reqCount.m.Lock()
        reqCount.c--
        reqCount.m.Unlock()
        return w.Conn.Close()
}

type SupervisingListener struct {
        net.Listener
}

func (sl *SupervisingListener) Accept() (c net.Conn, err error) {
        c, err = sl.Listener.Accept()
        if err != nil {
                return
        }
        c = SupervisedConn{Conn: c}
        log.Printf("open on conn to %v", c.RemoteAddr())
        reqCount.m.Lock()
        reqCount.c++
        reqCount.m.Unlock()
        return
}

// Block this goroutine awaiting signals.  With the exception of SIGTERM
// taking the place of SIGQUIT, signals are handled exactly as in Nginx
// and Unicorn: <http://unicorn.bogomips.org/SIGNALS.html>.
func AwaitSignals(l *net.UnixListener) error {
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGUSR2)
	for {
		sig := <-ch
		log.Println(sig.String())
		switch sig {

		// TODO SIGHUP should reload configuration.

		// SIGQUIT should exit gracefully.  However, Go doesn't seem
		// to like handling SIGQUIT (or any signal which dumps core by
		// default) at all so SIGTERM takes its place.  How graceful
		// this exit is depends on what the program does after this
		// function returns control.
		case syscall.SIGTERM:
			return nil

		// TODO SIGUSR1 should reopen logs.

		// SIGUSR2 begins the process of restarting without dropping
		// the listener passed to this function.
		case syscall.SIGUSR2:
                       log.Print("Got relaunch signal.")

			err := Relaunch(l)
			if nil != err {
				return err
			}
                       log.Print("Child launched")
                       f, err := l.File()
                       if nil != err {
                               return err
                       }
                       err = fclose(int(f.Fd()))
                       if nil != err {
                               return err
                       }
                       log.Printf("Server no longer accepting requests.  Outstanding requests: %d", reqCount.get())

                        for i := 0; (i < 10) && reqCount.get() > 0 ; i++ {
                                log.Printf("waiting for %d ongoing requests...", reqCount.get())
                                time.Sleep(1 * time.Second)
                        }

                        if reqCount.get() == 0 {
                                log.Print("server gracefully stopped.")
                                os.Exit(0)
                        } else {
                                log.Fatalf("server stopped after 10 seconds with %d clients still connected.", reqCount.get())
                        }

		}
	}
	return nil // It'll never get here.
}

// Convert and validate the GOAGAIN_FD environment
// variable.  If both are present and in order, this is a child process
// that may pick up where the parent left off.
func GetEnvs() (*net.UnixListener, error) {
	envFd := os.Getenv("GOAGAIN_FD")
	if "" == envFd {
		return nil, errors.New("GOAGAIN_FD not set")
	}
	var fd uintptr
	_, err := fmt.Sscan(envFd, &fd)
	if nil != err {
		return nil, err
	}
	tmp, err := net.FileListener(os.NewFile(fd, "listener"))
	if nil != err {
		return nil, err
	}
	l := tmp.(*net.UnixListener)
	return l, nil
}

// Re-exec this image without dropping the listener passed to this function.
func Relaunch(l *net.UnixListener) error {
	f, err := l.File()
	if nil != err {
		return err
	}
	noCloseOnExec(f.Fd())
	argv0, err := exec.LookPath(os.Args[0])
	if nil != err {
		return err
	}
	wd, err := os.Getwd()
	if nil != err {
		return err
	}
	err = os.Setenv("GOAGAIN_FD", fmt.Sprint(f.Fd()))
	if nil != err {
		return err
	}
	p, err := os.StartProcess(argv0, os.Args, &os.ProcAttr{
		Dir:   wd,
		Env:   os.Environ(),
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr, f},
		Sys:   &syscall.SysProcAttr{},
	})
	if nil != err {
		return err
	}
	log.Printf("spawned child %d\n", p.Pid)
	return nil
}

// Taken from upgradable.go

// These are here because there is no API in syscall for turning OFF
// close-on-exec (yet).

// from syscall/zsyscall_linux_386.go, but it seems like it might work
// for other platforms too.
func fcntl(fd int, cmd int, arg int) (val int, err error) {
        if runtime.GOOS != "linux" {
                log.Fatal("Function fcntl has not been tested on other platforms than linux.")
        }

        r0, _, e1 := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), uintptr(cmd), uintptr(arg))
        val = int(r0)
        if e1 != 0 {
                err = e1
        }
        return
}

func noCloseOnExec(fd uintptr) {
        fcntl(int(fd), syscall.F_SETFD, ^syscall.FD_CLOEXEC)
}

func fclose(fd int) (err error) {
        if runtime.GOOS != "linux" {
                log.Fatal("Function fclose has not been tested on other platforms than linux.")
        }

        err = syscall.Close(fd)
        return
}

func ListenAndServe(proto string, addr string) {
    // FIXME: support UNIX sockets (proto unix)
        l, err := GetEnvs()

        if nil != err {

                log.Printf("Opening socket for the first time because %s", err)
                // Listen on a TCP socket and accept connections in a new goroutine.
                laddr, err := net.ResolveUnixAddr(proto, addr)
                if nil != err {
                        log.Println(err)
                        os.Exit(1)
                }
                log.Printf("listening on %v", laddr)
                l, err = net.ListenUnix(proto, laddr)
                if nil != err {
                        log.Println(err)
                        os.Exit(1)
                }
                m := &SupervisingListener{Listener: l}
                go http.Serve(m, nil)

        } else {

                // Resume listening and accepting connections in a new goroutine.
                log.Printf("resuming listening on %v", l.Addr())
                m := &SupervisingListener{Listener: l}
                go http.Serve(m, nil)

        }

        // Block the main goroutine awaiting signals.
        if err := AwaitSignals(l); nil != err {
                log.Println(err)
                os.Exit(1)
        }
}
