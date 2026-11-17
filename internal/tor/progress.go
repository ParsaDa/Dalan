package tor

import (
	"regexp"
	"strconv"
	"sync"
)

var bootstrapLine = regexp.MustCompile(`Bootstrapped (\d{1,3})%`)

// ParseBootstrap reads how far Tor has got from one of its log lines.
func ParseBootstrap(line string) (percent int, ok bool) {
	match := bootstrapLine.FindStringSubmatch(line)
	if match == nil {
		return 0, false
	}
	percent, err := strconv.Atoi(match[1])
	if err != nil || percent > 100 {
		return 0, false
	}
	return percent, true
}

// Progress remembers the last bootstrap percentage each country's Tor printed.
type Progress struct {
	mu      sync.Mutex
	percent map[string]int
}

func (p *Progress) Set(country string, percent int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.percent == nil {
		p.percent = map[string]int{}
	}
	p.percent[country] = percent
}

func (p *Progress) Get(country string) (int, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	percent, ok := p.percent[country]
	return percent, ok
}

func (p *Progress) Reset(country string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.percent, country)
}
