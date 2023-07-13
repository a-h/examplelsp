package protocol

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/textproto"
	"strconv"
	"sync"

	"golang.org/x/exp/slog"
)

type Message struct {
	// ProtocolVersion is a string specifying the version of the JSON-RPC protocol. MUST be exactly "2.0".
	ProtocolVersion string `json:"jsonrpc"`
	// ID is an identifier established by the Client that MUST contain a String, Number, or NULL value if included. If it is not included it is assumed to be a notification. The value SHOULD normally not be Null [1] and Numbers SHOULD NOT contain fractional parts [2]
	ID *json.RawMessage `json:"id"`
}

func (msg Message) IsNotification() bool {
	return msg.ID == nil
}

type Request struct {
	Message
	// Method is a string containing the name of the method to be invoked. Method names that begin with the word rpc followed by a period character (U+002E or ASCII 46) are reserved for rpc-internal methods and extensions and MUST NOT be used for anything else.
	Method string `json:"method"`
	// Params is a structured value that holds the parameter values to be used during the invocation of the method. This member MAY be omitted.
	Params json.RawMessage `json:"params"`
}

type Response struct {
	Message
	// Result is populated on success.
	// This member is REQUIRED on success.
	// This member MUST NOT exist if there was an error invoking the method.
	// The value of this member is determined by the method invoked on the Server.
	Result any `json:"result"`
	// Error is populated on failure.
	// This member is REQUIRED on error.
	// This member MUST NOT exist if there was no error triggered during invocation.
	Error *Error `json:"error"`
}

func newError(err error) *Error {
	if err != nil {
		return nil
	}
	if e, isError := err.(*Error); isError {
		return e
	}
	return &Error{
		Code:    0,
		Message: err.Error(),
		Data:    nil,
	}
}

type Error struct {
	// Code is a Number that indicates the error type that occurred.
	Code int64 `json:"code"`
	// Message of the error.
	// The message SHOULD be limited to a concise single sentence.
	Message string `json:"message"`
	// A Primitive or Structured value that contains additional information about the error.
	// This may be omitted.
	// The value of this member is defined by the Server (e.g. detailed error information, nested errors etc.).
	Data any `json:"data"`
}

func (e *Error) Error() string {
	return e.Message
}

var (
	ErrParseError     *Error = &Error{Code: -32700, Message: "Parse error"}
	ErrInvalidRequest *Error = &Error{Code: -32600, Message: "Invalid Request"}
	ErrMethodNotFound *Error = &Error{Code: -32601, Message: "Method not found"}
	ErrInvalidParams  *Error = &Error{Code: -32602, Message: "Invalid params"}
	ErrInternal       *Error = &Error{Code: -32603, Message: "Internal error"}
)

func Read(r *bufio.Reader) (req Request, err error) {
	// Read header.
	header, err := textproto.NewReader(r).ReadMIMEHeader()
	if err != nil {
		return
	}
	contentLength, err := strconv.ParseInt(header.Get("Content-Length"), 10, 64)
	if err != nil {
		return req, ErrInvalidContentLengthHeader
	}
	// Read body.
	err = json.NewDecoder(io.LimitReader(r, contentLength)).Decode(&req)
	return
}

var ErrInvalidContentLengthHeader = errors.New("missing or invalid Content-Length header")

func Write(w *bufio.Writer, resp Response) (err error) {
	// Calculate body size.
	body, err := json.Marshal(resp)
	if err != nil {
		return
	}
	// Write the header.
	_, err = w.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body)))
	if err != nil {
		return
	}
	// Write the body.
	_, err = w.Write(body)
	if err != nil {
		return
	}
	// Flush.
	err = w.Flush()
	return
}

type Transport struct {
	r                    *bufio.Reader
	concurrencyLimit     int64
	notificationHandlers map[string]NotificationHandler
	requestHandlers      map[string]RequestHandler
	w                    *bufio.Writer
	writeLock            *sync.Mutex
	log                  *slog.Logger
	error                func(err error)
}

func (t *Transport) Notify(resp Response) (err error) {
	t.writeLock.Lock()
	defer t.writeLock.Unlock()
	return Write(t.w, resp)
}

func (t *Transport) Handle() (err error) {
	sem := make(chan struct{}, t.concurrencyLimit)
	for {
		sem <- struct{}{}
		req, err := Read(t.r)
		if err != nil {
			return err
		}
		go func(req Request) {
			t.handleRequest(req)
			<-sem
		}(req)
	}
}

func (t *Transport) handleRequest(req Request) {
	log := t.log.With(slog.Any("id", req.ID), slog.String("method", req.Method))
	if req.IsNotification() {
		handler, ok := t.notificationHandlers[req.Method]
		if !ok {
			log.Warn("notification not handled")
			return
		}
		// We don't need to notify clients if the notification results in an error.
		if err := handler.Handle(req.Params); err != nil && t.error != nil {
			log.Error("failed to handle notification", slog.Any("error", err))
			t.error(err)
		}
		return
	}
	//TODO: Handle batch requests?
	handler, ok := t.requestHandlers[req.Method]
	if !ok {
		log.Error("method not found")
		if err := t.Notify(Response{
			Message: Message{ProtocolVersion: "2.0", ID: req.ID},
			Error:   ErrMethodNotFound,
		}); err != nil {
			log.Error("failed to respond", slog.Any("error", err))
			t.error(fmt.Errorf("failed to respond: %w", err))
		}
		return
	}
	res := Response{
		Message: Message{ProtocolVersion: "2.0", ID: req.ID},
	}
	result, err := handler.Handle(req.Params)
	if err != nil {
		log.Error("failed to handle", slog.Any("error", err))
		res.Error = newError(err)
		result = nil
	}
	res.Result = result
	if err = t.Notify(res); err != nil {
		log.Error("failed to respond", slog.Any("error", err))
		t.error(fmt.Errorf("failed to respond: %w", err))
	}
}

type RequestHandler interface {
	Method() string
	Handle(params json.RawMessage) (result any, err error)
}

type NotificationHandler interface {
	Method() string
	Handle(params json.RawMessage) (err error)
}
