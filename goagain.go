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
	"reflect"
	"syscall"
)

var (
	// OnSIGHUP is the function called when the server receives a SIGHUP
	// signal. The normal use case for SIGHUP is to reload the
	// configuration.
	OnSIGHUP func(l net.Listener) error

	// OnSIGUSR1 is the function called when the server receives a
	// SIGUSR1 signal. The normal use case for SIGUSR1 is to repon the
	// log files.
	OnSIGUSR1 func(l net.Listener) error
)

// Export an error equivalent to net.errClosing for use with Accept during
// a graceful exit.
var ErrClosing = errors.New("use of closed network connection")

// Block this goroutine awaiting signals.  With the exception of SIGTERM
// taking the place of SIGQUIT, signals are handled exactly as in Nginx
// and Unicorn: <http://unicorn.bogomips.org/SIGNALS.html>.
func AwaitSignals(l net.Listener) error {
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGUSR2)
	for {
		sig := <-ch
		log.Println(sig.String())
		switch sig {

		// SIGHUP should reload configuration.
		case syscall.SIGHUP:
			if OnSIGHUP != nil {
				err := OnSIGHUP(l)
				if err != nil {
					log.Println("result of OnSigHup:", err)
				}
			}

		// SIGQUIT should exit gracefully.
		case syscall.SIGQUIT:
			return nil

		// SIGTERM should exit.
		case syscall.SIGTERM:
			return nil

		// SIGUSR1 should reopen logs.
		case syscall.SIGUSR1:
			if OnSIGUSR1 != nil {
				err := OnSIGUSR1(l)
				if err != nil {
					log.Println("result of OnSigUsr1:", err)
				}
			}

		// SIGUSR2 begins the process of restarting without dropping
		// the listener passed to this function.
		case syscall.SIGUSR2:
			err := Relaunch(l)
			if nil != err {
				log.Println(err)
			}

		}
	}

	return nil
}

// Convert and validate the GOAGAIN_FD, GOAGAIN_NAME, and GOAGAIN_PPID
// environment variables.  If all three are present and in order, this
// is a child process that may pick up where the parent left off.
func GetEnvs() (l net.Listener, ppid int, err error) {
	var fd uintptr
	_, err = fmt.Sscan(os.Getenv("GOAGAIN_FD"), &fd)
	if nil != err {
		return
	}
	var i net.Listener
	i, err = net.FileListener(os.NewFile(fd, os.Getenv("GOAGAIN_NAME")))
	if nil != err {
		return
	}
	switch i.(type) {
	case *net.TCPListener:
		l = i.(*net.TCPListener)
	case *net.UnixListener:
		l = i.(*net.UnixListener)
	default:
		err = errors.New(fmt.Sprintf(
			"file descriptor is %T not *net.TCPListener or *net.UnixListener",
			i,
		))
		return
	}
	if err = syscall.Close(int(fd)); nil != err {
		return
	}
	_, err = fmt.Sscan(os.Getenv("GOAGAIN_PPID"), &ppid)
	if nil != err {
		return
	}
	if syscall.Getppid() != ppid {
		err = errors.New(fmt.Sprintf(
			"GOAGAIN_PPID is %d but parent is %d",
			ppid,
			syscall.Getppid(),
		))
		return
	}
	return
}

// Send SIGQUIT to the given ppid in order to complete the handoff to the
// child process.
func KillParent(ppid int) error {
	return syscall.Kill(ppid, syscall.SIGQUIT)
}

// Re-exec this image without dropping the listener passed to this function.
func Relaunch(l net.Listener) error {
	argv0, err := exec.LookPath(os.Args[0])
	if nil != err {
		return err
	}
	if _, err := os.Stat(argv0); nil != err {
		return err
	}
	wd, err := os.Getwd()
	if nil != err {
		return err
	}
	v := reflect.ValueOf(l).Elem().FieldByName("fd").Elem()
	fd := uintptr(v.FieldByName("sysfd").Int())
	if err := os.Setenv("GOAGAIN_FD", fmt.Sprint(fd)); nil != err {
		return err
	}
	if err := os.Setenv("GOAGAIN_NAME", fmt.Sprintf("tcp:%s->", l.Addr().String())); nil != err {
		return err
	}
	if err := os.Setenv("GOAGAIN_PPID", fmt.Sprint(syscall.Getpid())); nil != err {
		return err
	}
	files := make([]*os.File, fd+1)
	files[syscall.Stdin] = os.Stdin
	files[syscall.Stdout] = os.Stdout
	files[syscall.Stderr] = os.Stderr
	files[fd] = os.NewFile(fd, string(v.FieldByName("sysfile").String()))
	p, err := os.StartProcess(argv0, os.Args, &os.ProcAttr{
		Dir:   wd,
		Env:   os.Environ(),
		Files: files,
		Sys:   &syscall.SysProcAttr{},
	})
	if nil != err {
		return err
	}
	log.Printf("spawned child %d\n", p.Pid)
	return nil
}
