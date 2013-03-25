package main

import (
	"goagain"
	"net/http"
	"fmt"
	"time"
)

func main() {

        http.HandleFunc("/hello", HelloServer)
        http.HandleFunc("/slow", WaitFive)
        http.HandleFunc("/superslow", WaitFifteen)
        goagain.ListenAndServe("unix", "/tmp/goagain-example.sock")

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
