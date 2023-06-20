package main

import (
	"context"
	"crypto/rand"
	"flag"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/hashicorp/yamux"
	"github.com/lainio/err2/try"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/experimental/gojs"
)

func main() {
	var callSystemExec bool
	flag.BoolVar(&callSystemExec, "sys", false, "use system exec")
	flag.Parse()

	cmdIn, cmdWriter := io.Pipe()
	cmdReader, cmdOut := io.Pipe()

	stdio := Stdio{
		PipeReader: cmdReader,
		PipeWriter: cmdWriter,
	}

	if callSystemExec { // run native
		cmd := exec.Command("go", "run", "./w")
		cmd.Stdin = cmdIn
		cmd.Stdout = cmdOut
		cmd.Stderr = os.Stderr
		try.To(cmd.Start())
		defer cmd.Wait()
	} else { // run wasm
		ctx := context.Background()
		rtc := wazero.NewRuntimeConfigInterpreter()
		rt := wazero.NewRuntimeWithConfig(ctx, rtc)
		gojs.MustInstantiate(ctx, rt)

		// build wasm
		cmd := exec.Command("go", "build", "-o", "w.wasm", "./w")
		cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
		cmd.Stderr = os.Stderr
		try.To(cmd.Run())

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
			mcc := gojs.NewConfig(mc)
			try.To(gojs.Run(context.Background(), rt, cm, mcc))
		}()
	}

	session := try.To1(yamux.Client(stdio, nil))
	defer session.Close()

	log.Println("ping")
	try.To1(session.Ping())
	log.Println("pong")

}

type Stdio struct {
	*io.PipeReader
	*io.PipeWriter
}

var _ io.ReadWriteCloser = (*Stdio)(nil)

func (o Stdio) Write(p []byte) (n int, err error) {
	log.Println("write", p)
	n, err = o.PipeWriter.Write(p)
	log.Println("wrote", n, err)
	return
}

func (o Stdio) Read(p []byte) (n int, err error) {
	log.Println("read")
	n, err = o.PipeReader.Read(p)
	log.Println("readed", n, err, p[:n])
	return
}

func (o Stdio) Close() (err error) {
	if err = o.PipeReader.Close(); err != nil {
		return
	}
	if err = o.PipeWriter.Close(); err != nil {
		return
	}
	return nil
}
