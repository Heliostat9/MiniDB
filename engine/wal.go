package engine

import (
	"bufio"
	"os"
	"strings"
	"sync"
)

const walFile = "data.wal"

var (
	walMu     sync.Mutex
	walReplay bool
)

func appendWAL(entry string) error {
	if walReplay {
		return nil
	}
	if txCtx != nil {
		txCtx.wal = append(txCtx.wal, entry)
		return nil
	}
	walMu.Lock()
	defer walMu.Unlock()
	f, err := os.OpenFile(walFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(entry + "\n"); err != nil {
		return err
	}
	return nil
}

func clearWAL() error {
	if walReplay {
		return nil
	}
	if txCtx != nil {
		return nil
	}
	walMu.Lock()
	defer walMu.Unlock()
	if err := os.Remove(walFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func replayWAL() error {
	walMu.Lock()
	data, err := os.ReadFile(walFile)
	walMu.Unlock()
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	walReplay = true
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if _, err := HandleCommand(line); err != nil {
			walReplay = false
			return err
		}
	}
	walReplay = false

	walMu.Lock()
	err = os.Remove(walFile)
	walMu.Unlock()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return scanner.Err()
}
