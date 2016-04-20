package main

import (
  "io"
  "fmt"
  "bufio"
  "bytes"
  "errors"
  log "github.com/golang/glog"
)

var (
	GREETING_ABI_V2 = []byte("HELO\t2")
	GREETING_REPLY  = "OK\tpowerdns-consul\n"
	END_REPLY       = "END\n"
	FAIL_REPLY      = "FAIL\n"
  PONG_REPLY      = "PONG\n"
)

const (
	KIND_AXFR = "AXFR"
	KIND_Q    = "Q"
	KIND_PING = "PING"
)

type PdnsRequest struct {
  kind     string
  qname    string
  qclass   string
  qtype    string
  id       string
  remoteIp string
  localIp  string
}

type PdnsResponse struct {
  qname   string
  qclass  string
  qtype   string
  ttl     string
  id      string
  content string
}

var (
	errLongLine = errors.New("pdns line too long")
	errBadLine  = errors.New("pdns line unparseable")
)

type PowerDNSHandler struct {
  lookupCallback func(request *PdnsRequest) (responses []*PdnsResponse, err error)
}

func (h *PowerDNSHandler) parseRequest(line []byte) (request *PdnsRequest, err error) {
  tokens := bytes.Split(line, []byte("\t"))
  kind := string(tokens[0])

  switch kind {
  case KIND_Q:
    if len(tokens) < 7 {
      return nil, errBadLine
    }
    return &PdnsRequest{kind, string(tokens[1]), string(tokens[2]), string(tokens[3]), string(tokens[4]), string(tokens[5]), string(tokens[6])}, nil
  case KIND_PING, KIND_AXFR:
    return &PdnsRequest{kind: kind}, nil
  default:
    return nil, errBadLine
  }
}

func (h *PowerDNSHandler) formatResponse(resp *PdnsResponse) (lines string) {
  return fmt.Sprintf("DATA\t%v\t%v\t%v\t%v\t%v\t%v\n", resp.qname, resp.qclass, resp.qtype, resp.ttl, resp.id, resp.content)
}

func (h *PowerDNSHandler) write(out io.Writer, line string) (err error) {
  _, err = io.WriteString(out, line)
  return err
}

func (h *PowerDNSHandler) Handle(in io.Reader, out io.Writer) {
  log.Infof("Started Handler")
  bufReader := bufio.NewReader(in)
  handshakeReceived := false

  for {
    line, isPrefix, err := bufReader.ReadLine()

    if isPrefix {
      log.Errorf("Got a prefixed line, returning")
      return
    }

    if err != nil {
      log.Errorf("Error reading line: %v", err)
    }

    if !handshakeReceived {
      if !bytes.Equal(line, GREETING_ABI_V2) {
        log.Errorf("Handshake failed: %s != %s", line, GREETING_ABI_V2)
        h.write(out, FAIL_REPLY)
      } else {
        handshakeReceived = true
        h.write(out, GREETING_REPLY)
      }

      continue
    }

    request, err := h.parseRequest(line)
    if err != nil {
      log.Errorf("Failed parsing request: %v", err)
      h.write(out, FAIL_REPLY)
      continue
    }

    switch request.kind {
    case KIND_Q:
      responses, err := lookup(request)
      if err != nil {
        log.Errorf("Query for %v failed: %v", request.qname, err)
        h.write(out, FAIL_REPLY)
        continue
      }

      for _, response := range responses {
        h.write(out, h.formatResponse(response))
      }
    case KIND_AXFR:
      // not implemented
    case KIND_PING:
      h.write(out, PONG_REPLY)
    }

    h.write(out, END_REPLY)
  }
}
