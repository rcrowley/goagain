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
	for {
		sig := <-signal.Incoming
		log.Println(sig.String())
		if unixSig, ok := sig.(os.UnixSignal); ok {
			switch unixSig {

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

			// SIGTSTP escalates to the unblockable SIGSTOP in order to
			// provide familiar Ctrl+Z semantics in terminals.
			case syscall.SIGTSTP:
				err := syscall.Kill(syscall.Getpid(), syscall.SIGSTOP)
				if nil != err {
					return err
				}

			// Other signals exit immediately.
			default:
				os.Exit(128 + int(unixSig))

			}
		}
	}
	return nil // It'll never get here.
}

// Convert and validate the GOAGAIN_FD and GOAGAIN_PPID environment
// variables.  If both are present and in order, this is a child process
// that may pick up where the parent left off.
func GetEnvs() (*net.TCPListener, int, error) {
	envFd, err := os.Getenverror("GOAGAIN_FD")
	if nil != err {
		return nil, 0, err
	}
	fd, err := strconv.Atoi(envFd)
	if nil != err {
		return nil, 0, err
	}
	tmp, err := net.FileListener(os.NewFile(fd, "listener"))
	if nil != err {
		return nil, 0, err
	}
	l := tmp.(*net.TCPListener)
	envPpid, err := os.Getenverror("GOAGAIN_PPID")
	if nil != err {
		return l, 0, err
	}
	ppid, err := strconv.Atoi(envPpid)
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
	err = os.Setenv("GOAGAIN_FD", strconv.Itoa(f.Fd()))
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
