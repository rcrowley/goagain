package main

import (
	"goagain"
	"log"
	"net/http"
	"fmt"
	"time"
	"syscall"
)

func main() {

        log.SetPrefix(fmt.Sprintf("[%5d] ", syscall.Getpid()))
        http.HandleFunc("/hello", HelloServer)
        http.HandleFunc("/slow", WaitFive)
        http.HandleFunc("/superslow", WaitFifteen)
        goagain.ListenAndServe("tcp", "127.0.0.1:48879")

}

func HelloServer(w http.ResponseWriter, req *http.Request) {
        fmt.Fprintf(w, "hello world\n")
}

func WaitFive(w http.ResponseWriter, req *http.Request) {
        time.Sleep(5 * time.Second)
        fmt.Fprintf(w, "sorry for being slow\n")
}

func WaitFifteen(w http.ResponseWriter, req *http.Request) {
        time.Sleep(15 * time.Second)
        fmt.Fprintf(w, "sorry for being slow\n")
}
