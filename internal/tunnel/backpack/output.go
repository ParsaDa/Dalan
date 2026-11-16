package backpack

import (
	"bytes"
	"strings"
	"sync"

	"github.com/mhsanaei/3x-ui/v3/internal/logger"
)

const (
	outputTailLines = 10
	outputLineLimit = 500
)

// outputLog turns an engine's stdout and stderr into log lines and remembers the last few for a crash report.
type outputLog struct {
	emit func(line string)

	mu   sync.Mutex
	part []byte
	tail []string
}

func (o *outputLog) Write(p []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.part = append(o.part, p...)
	for {
		i := bytes.IndexByte(o.part, '\n')
		if i < 0 {
			break
		}
		o.line(string(o.part[:i]))
		o.part = o.part[i+1:]
	}
	if len(o.part) > outputLineLimit*4 {
		o.line(string(o.part))
		o.part = nil
	}
	return len(p), nil
}

func (o *outputLog) line(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if len(text) > outputLineLimit {
		text = text[:outputLineLimit]
	}
	o.tail = append(o.tail, text)
	if len(o.tail) > outputTailLines {
		o.tail = o.tail[1:]
	}
	o.emit(text)
}

// finish flushes an unterminated last line and returns what the engine said last.
func (o *outputLog) finish() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	if len(o.part) > 0 {
		o.line(string(o.part))
		o.part = nil
	}
	return strings.Join(o.tail, " | ")
}

func (s *Supervisor) outputFor(name string) *outputLog {
	emit := s.LogLine
	if emit == nil {
		emit = func(tunnel, line string) { logger.Infof("BackPack %s: %s", tunnel, line) }
	}
	return &outputLog{emit: func(line string) { emit(name, line) }}
}
