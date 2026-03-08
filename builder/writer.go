package builder

import (
	"bytes"
	"fmt"
	"os"
)

// prefixWriter wraps os.Stderr and prepends a prefix to each line.
type prefixWriter struct {
	prefix string
	buf    bytes.Buffer
}

func newPrefixWriter(prefix string) *prefixWriter {
	return &prefixWriter{prefix: prefix}
}

func (p *prefixWriter) Write(data []byte) (int, error) {
	n := len(data)
	p.buf.Write(data)
	for {
		idx := bytes.IndexByte(p.buf.Bytes(), '\n')
		if idx < 0 {
			break
		}
		line := p.buf.Next(idx + 1)
		fmt.Fprintf(os.Stderr, "%s%s", p.prefix, line)
	}
	return n, nil
}
