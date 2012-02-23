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
	"strconv"
	"syscall"
)

// Block this goroutine awaiting signals.  With the exception of SIGTERM
// taking the place of SIGQUIT, signals are handled exactly as in Nginx
// and Unicorn: <http://unicorn.bogomips.org/SIGNALS.html>.
func AwaitSignals(l *net.TCPListener) error {
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
			err := Relaunch(l)
			if nil != err {
				return err
			}

		}
	}
	return nil // It'll never get here.
}

// Convert and validate the GOAGAIN_FD and GOAGAIN_PPID environment
// variables.  If both are present and in order, this is a child process
// that may pick up where the parent left off.
func GetEnvs() (*net.TCPListener, int, error) {
	envFd := os.Getenv("GOAGAIN_FD")
	if "" == envFd {
		return nil, 0, errors.New("GOAGAIN_FD not set")
	}
	var fd uintptr
	_, err := fmt.Sscan(envFd, fd)
	if nil != err {
		return nil, 0, err
	}
	tmp, err := net.FileListener(os.NewFile(fd, "listener"))
	if nil != err {
		return nil, 0, err
	}
	l := tmp.(*net.TCPListener)
	envPpid := os.Getenv("GOAGAIN_PPID")
	if "" == envPpid {
		return l, 0, errors.New("GOAGAIN_PPID not set")
	}
	var ppid int
	_, err = fmt.Sscan(envPpid, ppid)
	if nil != err {
		return l, 0, err
	}
	if syscall.Getppid() != ppid {
		return l, ppid, errors.New(fmt.Sprintf(
			"GOAGAIN_PPID is %d but parent is %d\n", ppid, syscall.Getppid()))
	}
	return l, ppid, nil
}

// Send SIGQUIT (but really SIGTERM since Go can't handle SIGQUIT) to the
// given ppid in order to complete the handoff to the child process.
func KillParent(ppid int) error {
	err := syscall.Kill(ppid, syscall.SIGTERM)
	if nil != err {
		return err
	}
	return nil
}

// Re-exec this image without dropping the listener passed to this function.
func Relaunch(l *net.TCPListener) error {
	f, err := l.File()
	if nil != err {
		return err
	}
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
	err = os.Setenv("GOAGAIN_PPID", strconv.Itoa(syscall.Getpid()))
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
