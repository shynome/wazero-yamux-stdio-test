package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"

	"github.com/hashicorp/yamux"
	"github.com/lainio/err2/try"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

func main() {
	var callSystemExec bool
	flag.BoolVar(&callSystemExec, "sys", false, "use system exec")
	var useWasmer bool
	flag.BoolVar(&useWasmer, "wasmer", false, "use wasm3 call")
	var useGoJS bool
	flag.BoolVar(&useGoJS, "gojs", false, "use gojs call")
	flag.Parse()

	cmdIn, cmdWriter := try.To2(os.Pipe())
	cmdReader, cmdOut := try.To2(os.Pipe())

	stdio := Stdio{
		Reader: cmdReader,
		Writer: cmdWriter,
	}

	if callSystemExec { // run native
		cmd := exec.Command("go", "run", "./w")
		cmd.Stdin = cmdIn
		cmd.Stdout = cmdOut
		cmd.Stderr = os.Stderr
		try.To(cmd.Start())
		defer cmd.Wait()
	} else if useWasmer {
		// build wasip1
		build := exec.Command("gotip", "build", "-o", "w.wasm", "./w")
		build.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
		build.Stderr = os.Stderr
		try.To(build.Run())

		cmd := exec.Command("wasmer", "run", "w.wasm")
		cmd.Stdin = cmdIn
		cmd.Stdout = cmdOut
		cmd.Stderr = os.Stderr
		try.To(cmd.Start())
		defer cmd.Wait()
	} else if useGoJS {
		gojsExec := fmt.Sprintf("%s/misc/wasm/go_js_wasm_exec", runtime.GOROOT())
		cmd := exec.Command("go", "run", "-exec", gojsExec, "./w")
		cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
		cmd.Stdin = cmdIn
		cmd.Stdout = cmdOut
		cmd.Stderr = os.Stderr
		try.To(cmd.Start())
		defer cmd.Wait()
	} else { // run gotip wasm
		// build gotip wasm
		build := exec.Command("gotip", "build", "-o", "w.wasm", "./w")
		build.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
		build.Stderr = os.Stderr
		try.To(build.Run())

		ctx := context.Background()
		rtc := wazero.NewRuntimeConfigInterpreter()
		rt := wazero.NewRuntimeWithConfig(ctx, rtc)
		wasi_snapshot_preview1.MustInstantiate(ctx, rt)

		b := try.To1(os.ReadFile("./w.wasm"))
		cm := try.To1(rt.CompileModule(ctx, b))
		mc := wazero.NewModuleConfig().
			WithArgs("wazero").
			WithRandSource(rand.Reader).
			WithSysNanosleep().
			WithSysNanotime().
			WithSysWalltime().
			WithStdin(cmdIn).
			WithStdout(cmdOut).
			WithStderr(os.Stderr)
		w := make(chan any)
		defer func() { <-w }()
		go func() {
			defer close(w)
			log.Println("run")
			rt.InstantiateModule(ctx, cm, mc)
		}()
	}

	session := try.To1(yamux.Client(stdio, nil))
	defer session.Close()

	log.Println("ping")
	try.To1(session.Ping())
	log.Println("pong")

}

type Stdio struct {
	io.Reader
	io.Writer
}

var _ io.ReadWriteCloser = (*Stdio)(nil)

func (o Stdio) Write(p []byte) (n int, err error) {
	log.Println("write", p)
	n, err = o.Writer.Write(p)
	log.Println("wrote", n, err)
	return
}

func (o Stdio) Read(p []byte) (n int, err error) {
	log.Println("read")
	n, err = o.Reader.Read(p)
	log.Println("readed", n, err, p[:n])
	return
}

func (o Stdio) Close() (err error) {
	if closer, ok := o.Reader.(io.Closer); ok {
		closer.Close()
	}
	if closer, ok := o.Writer.(io.Closer); ok {
		closer.Close()
	}
	return nil
}
