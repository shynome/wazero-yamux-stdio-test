package main

import (
	"io"
	"log"
	"net/http"
	"os"

	"github.com/hashicorp/yamux"
	"github.com/lainio/err2/try"
)

func main() {

	stdio := NewStdio()
	session := try.To1(yamux.Server(stdio, nil))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello world")
	})

	http.Serve(session, nil)
}

type Stdio struct {
	io.Reader
	io.Writer
}

var _ io.ReadWriteCloser = (*Stdio)(nil)

func NewStdio() *Stdio {
	return &Stdio{
		Reader: os.Stdin,
		Writer: os.Stdout,
	}
}

func (o *Stdio) Read(p []byte) (n int, err error) {
	log.Println("wasm read")
	n, err = o.Reader.Read(p)
	log.Println("wasm readed", n, err, p[:n])
	return
}
func (o *Stdio) Write(p []byte) (n int, err error) {
	log.Println("wasm write", p)
	n, err = o.Writer.Write(p)
	log.Println("wasm wrote", n, err)
	return
}

func (Stdio) Close() error {
	os.Exit(1)
	return nil
}
