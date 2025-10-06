package gosmig

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"golang.org/x/term"
)

func runCmdStatus[
	TDBRow DBRow,
	TDBResult DBResult,
	TTX TX[TDBRow, TDBResult],
	TTXO TXOptions,
	TDB DB[TDBRow, TDBResult, TTX, TTXO]](

	ctx context.Context,
	migrations []Migration[TDBRow, TDBResult, TTX, TTXO, TDB],
	db TDB,
	output io.Writer,
	timeout time.Duration,
) error {

	sortMigrationsDesc(migrations)

	dbVersion, err := getDBVersion(ctx, db, timeout)
	if err != nil {
		return err
	}

	w := output
	if output == os.Stdout { // coverage-ignore
		var cleanupPager func() error
		w, cleanupPager = usePager()
		defer func() {
			if err := cleanupPager(); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "failed to cleanup pager: %v\n", err)
			}
		}()
	}

	_, _ = fmt.Fprintf(w, "%-10s %-12s\n", "VERSION", "STATUS")
	for _, migration := range migrations {
		status := "[ ] PENDING"
		if migration.Version <= dbVersion {
			status = "[x] APPLIED"
		}
		_, _ = fmt.Fprintf(w, "%-10d %-12s\n", migration.Version, status)
	}

	return nil
}

// usePager attempts to pipe output to $PAGER if available and stdout is a TTY.
// Otherwise it returns stdout.
func usePager() (io.Writer, func() error) { // coverage-ignore
	// Don't use pager if output is redirected/piped
	if !term.IsTerminal(int(syscall.Stdout)) {
		return os.Stdout, func() error { return nil }
	}

	pager := os.Getenv("PAGER")
	if pager == "" {
		// Fallback to common pagers
		pager = "less"
		if _, err := exec.LookPath(pager); err != nil {
			pager = "more"
			if _, err := exec.LookPath(pager); err != nil {
				// No pager available, use stdout
				return os.Stdout, func() error { return nil }
			}
		}
	}

	cmd := exec.Command(pager)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set better defaults for less if it's being used
	if pager == "less" || (pager != "more" && os.Getenv("LESS") == "") {
		// -F: exit if content fits on one screen
		// -R: allow ANSI color codes
		// -X: don't clear screen on exit
		// -K: exit on Ctrl-C
		cmd.Env = append(os.Environ(), "LESS=-FRX")
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return os.Stdout, func() error { return nil }
	}

	if err := cmd.Start(); err != nil {
		return os.Stdout, func() error { return nil }
	}

	cleanup := func() error {
		if err := stdin.Close(); err != nil {
			return fmt.Errorf("failed to close stdin: %w", err)
		}
		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("failed to wait for pager command: %w", err)
		}
		return nil
	}

	return stdin, cleanup
}
