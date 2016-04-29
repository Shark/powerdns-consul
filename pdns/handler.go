package pdns

import (
  "io"
  "fmt"
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

type Request struct {
  Kind     string
  Qname    string
  Qclass   string
  Qtype    string
  Id       string
  RemoteIp string
  LocalIp  string
}

type Response struct {
  Qname   string
  Qclass  string
  Qtype   string
  Ttl     string
  Id      string
  Content string
}

var (
  errLongLine = errors.New("pdns line too long")
  errBadLine  = errors.New("pdns line unparseable")
)

type Handler struct {
  Lookup func(request *Request) (responses []*Response, err error)
}

func (h *Handler) parseRequest(line []byte) (request *Request, err error) {
  tokens := bytes.Split(line, []byte("\t"))
  kind := string(tokens[0])

  switch kind {
  case KIND_Q:
    if len(tokens) < 7 {
      return nil, errBadLine
    }
    return &Request{kind, string(tokens[1]), string(tokens[2]), string(tokens[3]), string(tokens[4]), string(tokens[5]), string(tokens[6])}, nil
  case KIND_PING, KIND_AXFR:
    return &Request{Kind: kind}, nil
  default:
    return nil, errBadLine
  }
}

func (h *Handler) formatResponse(resp *Response) (lines string) {
  return fmt.Sprintf("DATA\t%v\t%v\t%v\t%v\t%v\t%v\n", resp.Qname, resp.Qclass, resp.Qtype, resp.Ttl, resp.Id, resp.Content)
}

func (h *Handler) write(out io.Writer, line string) (err error) {
  _, err = io.WriteString(out, line)
  return err
}

func (h *Handler) Handle(in chan []byte, out chan []byte) {
  log.Infof("Started Handler")
  handshakeReceived := false

  for {
    line := <-in

    if !handshakeReceived {
      if !bytes.Equal(line, GREETING_ABI_V2) {
        log.Errorf("Handshake failed: %s != %s", line, GREETING_ABI_V2)
        out <- []byte(FAIL_REPLY)
      } else {
        handshakeReceived = true
        out <- []byte(GREETING_REPLY)
      }

      continue
    }

    request, err := h.parseRequest(line)
    if err != nil {
      log.Errorf("Failed parsing request: %v", err)
      out <- []byte(FAIL_REPLY)
      continue
    }

    switch request.Kind {
    case KIND_Q:
      responses, err := h.Lookup(request)
      if err != nil {
        log.Errorf("Query for %v failed: %v", request.Qname, err)
        out <- []byte(FAIL_REPLY)
        continue
      }

      for _, response := range responses {
        out <- []byte(h.formatResponse(response))
      }
    case KIND_AXFR:
      // not implemented
    case KIND_PING:
      out <- []byte(PONG_REPLY)
    }

    out <- []byte(END_REPLY)
  }
}
