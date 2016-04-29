package pdns

import (
  "bytes"
  "errors"
  "reflect"
  "testing"
)

var parseRequestTests = []struct {
  request []byte
  expected *Request
}{
  {[]byte("Q\texample.invalid\tIN\tANY\t-1\t10.0.0.1\t127.0.0.1"), &Request{"Q", "example.invalid", "IN", "ANY", "-1", "10.0.0.1", "127.0.0.1"}},
  {[]byte("ABC\tDEF"), nil},
  {[]byte("Q\t\t\t\t\t\t"), &Request{"Q", "", "", "", "", "", ""}},
  {[]byte("Q\t\t\t\t\t"), nil},
  {[]byte("PING\t\t\t\t\t\t"), &Request{"PING", "", "", "", "", "", ""}},
  {[]byte("AXFR\t\t\t\t\t\t"), &Request{"AXFR", "", "", "", "", "", ""}},
  {[]byte(nil), nil},
  {[]byte(""), nil},
  {[]byte("PING\texample.invalid\tIN\tANY\t-1\t10.0.0.1\t127.0.0.1"), &Request{Kind: "PING"}},
  {[]byte("AXFR\texample.invalid\tIN\tANY\t-1\t10.0.0.1\t127.0.0.1"), &Request{Kind: "AXFR"}},
}

func TestParseRequest(t *testing.T) {
  handler := &Handler{nil}
  for _, tt := range parseRequestTests {
    actual, err := handler.parseRequest(tt.request)

    if err == nil {
      if !reflect.DeepEqual(actual, tt.expected) {
        t.Errorf("TestParseRequest: actual %v, expected %v", actual, tt.expected)
      }
    } else {
      if tt.expected != nil {
        t.Errorf("TestParseRequest: actual %v, did not expect error", err)
      }
    }
  }
}

var formatResponseTests = []struct {
  response *Response
  expected string
}{
  {&Response{"A", "B", "C", "D", "E", "F"}, "DATA\tA\tB\tC\tD\tE\tF\n"},
  {&Response{}, "DATA\t\t\t\t\t\t\n"},
}

func TestFormatResponse(t *testing.T) {
  handler := &Handler{nil}
  for _, tt := range formatResponseTests {
    actual := handler.formatResponse(tt.response)

    if actual != tt.expected {
      t.Errorf("TestFormatResponse: actual %s, expected %s", actual, tt.expected)
    }
  }
}

func handleLookupSuccess(request *Request) (responses []*Response, err error) {
  return []*Response{
    &Response{"A", "B", "C", "D", "E", "F"},
    &Response{"G", "H", "I", "J", "K", "L"},
    &Response{"M", "N", "O", "P", "Q", "R"},
  }, nil
}

func handleLookupFail(request *Request) (responses []*Response, err error) {
  return nil, errors.New("an error ^_^")
}

var handleTestsSuccess = []struct {
  sent []byte
  received []byte
}{
  {[]byte("HELO\t1"), []byte("FAIL\n")},
  {[]byte("ABC\t2"), []byte("FAIL\n")},
  {[]byte("HELO\t2"), []byte("OK\tpowerdns-consul\n")},
  {[]byte("AXFR\t\t\t\t\t\t"), []byte("END\n")},
  {[]byte("PING\t\t\t\t\t\t"), []byte("PONG\n")},
  {nil, []byte("END\n")},
  {[]byte("Q\tA\tB\tC\tD\tE\tF"), []byte("DATA\tA\tB\tC\tD\tE\tF\n")},
  {nil, []byte("DATA\tG\tH\tI\tJ\tK\tL\n")},
  {nil, []byte("DATA\tM\tN\tO\tP\tQ\tR\n")},
  {nil, []byte("END\n")},
}

var handleTestsFail = []struct {
  sent []byte
  received []byte
}{
  {[]byte("Q\tA\tB\tC\tD\tE\tF"), []byte("FAIL\n")},
  {[]byte("Q\tA\tB\tC\tD\tE\tF"), []byte("FAIL\n")},
}

func TestHandle(t *testing.T) {
  handler := &Handler{handleLookupSuccess}
  in, out := make(chan []byte), make(chan []byte)
  stageDone := make(chan bool)
  testDone := make(chan bool)
  go func() {
    handler.Handle(in, out)
  }()
  var (
    actual []byte
    numTests int
  )

  go func() {
    for _, tt := range handleTestsSuccess {
      numTests++
      if tt.sent != nil {
        in <- tt.sent
      }
      actual = <- out
      if !bytes.Equal(tt.received, actual) {
        t.Errorf("TestHandle stage success: actual %s, expected %s", actual, string(tt.received))
        testDone <- true
      }
    }

    stageDone <- true
  }()

  go func() {
    <- stageDone

    handler.Lookup = handleLookupFail
    for _, tt := range handleTestsFail {
      numTests++
      if tt.sent != nil {
        in <- tt.sent
      }
      actual = <- out
      if !bytes.Equal(tt.received, actual) {
        t.Errorf("TestHandle stage fail: actual %s, expected %s", actual, string(tt.received))
        testDone <- true
      }
    }

    testDone <- true
  }()

  <- testDone
  testCount := len(handleTestsSuccess) + len(handleTestsFail)
  if numTests != testCount {
    t.Errorf("TestHandle: %d tests were executed, expected %d", numTests, testCount)
  }
}
