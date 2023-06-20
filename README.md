## Problem description

`yamux` is working on `os/exec.Command.stdio`, but not working in `wazero wasi`.

it seem `goroutines` is not running at [`yamux/session.go#L636`](./yamux/session.go#L636),
I don't know why, can you help me?

thanks for your seen

## test

```sh
# you will see ping and pong output
go run . -sys
# you will only see ping, and the terminal is hang up
go run .
```

## Debug

add log at [`yamux/session.go#L636`](./yamux/session.go#L636), the log in `goroutines` is not running

```go
	if flags&flagSYN == flagSYN {
		log.Println("handle ping")
		go func() {
			log.Println("handle ping and send")
			hdr := header(make([]byte, headerSize))
			hdr.encode(typePing, flagACK, 0, pingID)
			if err := s.sendNoWait(hdr); err != nil {
				s.logger.Printf("[WARN] yamux: failed to send ping reply: %v", err)
			}
		}()
		return nil
	}
```
