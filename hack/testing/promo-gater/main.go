// promo-gater starts an HTTP server that gates Kargo promotion steps.
//
// Usage:
//
//	promo-gater [flags] [-- command [args...]]
//
// When a promotion's HTTP step sends a request to this server:
//   - With a command: executes the command, returns stdout (HTTP 200) or
//     stderr (HTTP 500) depending on the exit code.
//   - Without a command: blocks until the user presses Enter in the
//     terminal, then returns HTTP 200.
//
// The server exits after handling one request by default. Use --once=false
// to keep it running. See README.md for promotion step configuration examples.
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	port := flag.Int("port", 24365, "port to listen on")
	addr := flag.String("addr", "0.0.0.0", "bind address")
	once := flag.Bool("once", true, "exit after handling one request")
	flag.Parse()

	// Everything after flag parsing is the command to run.
	cmdArgs := flag.Args()

	listenAddr := fmt.Sprintf("%s:%d", *addr, *port)

	// Channel used by the handler to signal the server to shut down.
	done := make(chan struct{})

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf( //nolint:gosec
			"request received: %s %s from %s",
			r.Method, r.URL.Path, r.RemoteAddr,
		)

		// Pass request metadata as env vars for the command.
		body, _ := io.ReadAll(r.Body)
		env := append(
			os.Environ(),
			"GATE_METHOD="+r.Method,
			"GATE_PATH="+r.URL.Path,
			"GATE_QUERY="+r.URL.RawQuery,
			"GATE_BODY="+string(body),
		)

		if len(cmdArgs) > 0 {
			handleCommand(r.Context(), w, cmdArgs, env)
		} else {
			handleInteractive(w)
		}

		if *once {
			close(done)
		}
	})

	server := &http.Server{ // nolint: gosec
		Addr:              listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown on signal.
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer stop()

	go func() {
		select {
		case <-done:
		case <-ctx.Done():
		}
		_ = server.Shutdown(context.Background())
	}()

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("error listening on %s: %v", listenAddr, err)
	}
	if len(cmdArgs) > 0 {
		log.Printf(
			"listening on %s — will run: %s",
			ln.Addr(), strings.Join(cmdArgs, " "),
		)
	} else {
		log.Printf(
			"listening on %s — interactive mode (press Enter to release)",
			ln.Addr(),
		)
	}

	if err := server.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}
}

func handleCommand(
	ctx context.Context,
	w http.ResponseWriter,
	args []string,
	env []string,
) {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...) // nolint: gosec
	cmd.Env = env

	out, err := cmd.CombinedOutput()

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			log.Printf(
				"command exited with code %d",
				exitErr.ExitCode(),
			)
		} else {
			log.Printf("command error: %v", err)
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(out) // nolint: gosec
		return
	}

	log.Printf("command succeeded")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(out) // nolint: gosec
}

func handleInteractive(w http.ResponseWriter) {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(
		os.Stderr,
		">>> Promotion step is waiting. Press Enter to release...",
	)
	fmt.Fprintln(os.Stderr, "")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	log.Printf("gate released by user")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("released\n"))
}
