package edge

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/hysios/edgekv/utils"
	"github.com/hysios/log"
)

type FrameScanner struct {
	s *bufio.Scanner
}

func NewFrameScanner(rd io.ReadCloser) *FrameScanner {
	var (
		s       = bufio.NewScanner(rd)
		scanner = &FrameScanner{s: s}
	)

	s.Split(scanner.Split)
	return scanner
}

func (scanner *FrameScanner) Scan() bool {
	return scanner.s.Scan()
}

func (scanner *FrameScanner) Text() string {
	return scanner.s.Text()
}

func (scanner *FrameScanner) Split(data []byte, atEOF bool) (advance int, token []byte, err error) {
	p := bytes.IndexAny(data, "\n\n")
	if p > 0 {
		return p, data[:p], nil
	}

	return 0, nil, nil
}

func (scanner *FrameScanner) DecodeFrame() <-chan EdgeEvent {
	var ch = make(chan EdgeEvent)

	go func() {
		for scanner.Scan() {
			if event, err := scanner.decodeFrame(scanner.Text()); err != nil {
				log.Errorf("decodeFrame: decode error %s", err)
				continue
			} else {
				ch <- *event
			}
		}
	}()
	return ch
}

func (scanner *FrameScanner) decodeFrame(frame string) (*EdgeEvent, error) {
	var (
		ss    = strings.Split(frame, "change:")
		event = &EdgeEvent{}
		b     []byte
		err   error
	)

	raw := strings.TrimSpace(ss[1])
	log.Infof("ss: %v", raw)
	if b, err = base64.StdEncoding.DecodeString(raw); err != nil {
		return nil, fmt.Errorf("edge: decoder base64 error %s", err)
	}

	if err = utils.Unmarshal(b, event); err != nil {
		return nil, fmt.Errorf("edge: unmarshal error %s", err)
	}

	return event, nil
}
