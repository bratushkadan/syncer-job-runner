package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	updateInterval = time.Second
)

func runSyncer(ctx context.Context) <-chan struct{} {
	ch := make(chan struct{})

	go func() {
		defer close(ch)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if v := rand.Intn(10); v > 7 {
					ch <- struct{}{}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch
}

func runProcess(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)
		cmd := exec.CommandContext(ctx, "bash", "-c", `echo "starting subprocess program!" && while true; do echo "spam" && sleep 0.2; done`)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			var exerr *exec.ExitError
			if errors.Is(ctx.Err(), context.Canceled) && errors.As(err, &exerr) && exerr.ExitCode() == -1 {
				fmt.Println("canceled")
			} else {
				fmt.Println(fmt.Errorf("error executing bash command: %w", err))
			}
		}
	}()

	return done
}

func doBaseFor(fpath string) {
	dir := filepath.Dir(fpath)
	base := filepath.Base(fpath)
	fmt.Printf(`dir = "%s", base = "%s", path = "%s"`+"\n", dir, base, fpath)
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	syncCh := runSyncer(ctx)

	// run shit
	subctx, subcancel := context.WithCancel(ctx)
	procCh := runProcess(subctx)
	for {
		select {
		case <-procCh:
			if syncCh == nil {
				fmt.Println("end")
				return
			}
			subctx, subcancel = context.WithCancel(ctx)
			procCh = runProcess(subctx)
		case _, doneSyncer := <-syncCh:
			if !doneSyncer {
				syncCh = nil
			}
			subcancel()
		}
	}
}
