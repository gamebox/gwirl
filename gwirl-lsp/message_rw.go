package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"go.lsp.dev/jsonrpc2"
)

type LspMsgRW struct {
    rw *bufio.ReadWriter
    r *bufio.Reader
    w *bufio.Writer
    mutex *sync.Mutex
}

func NewMsgRWFromIO(reader io.Reader, writer io.Writer) (LspMsgRW, error) {
    if reader == nil || writer == nil {
        return LspMsgRW{}, errors.New("Can't create MsgRW with nil reader or writer")
    }
    r := bufio.NewReaderSize(reader, 40*1024)
    w := bufio.NewWriter(writer)
    rw := bufio.NewReadWriter(r, w)
    mutex := &sync.Mutex{}
    return LspMsgRW{
        rw,
        r,
        w,
        mutex,
    }, nil
}

func NewMsgRW(reader *bufio.Reader, writer *bufio.Writer) LspMsgRW {
    rw := bufio.NewReadWriter(reader, writer)
    mutex := &sync.Mutex{}
    return LspMsgRW{
        rw: rw,
        r: reader,
        w: writer,
        mutex: mutex,
    }
}

func (rw *LspMsgRW) WriteMessage(msg jsonrpc2.Message) error {
	bytes, err := json.Marshal(msg)
	if err != nil {
		return jsonrpc2.NewError(jsonrpc2.InternalError, fmt.Sprintf("Failed to write message msg=%v err=%v", msg, err))
	}
	length := len(bytes)

    rw.mutex.Lock()
	defer rw.mutex.Unlock()
	rw.rw.Write([]byte("Content-Length: "))
	rw.rw.Write([]byte(fmt.Sprintf("%d", length)))
	rw.rw.Write([]byte("\r\n\r\n"))
	rw.rw.Write(bytes)
    rw.rw.Flush()
    return nil
}

func (rw *LspMsgRW) ReadMessage() (jsonrpc2.Message, *jsonrpc2.Error) {
	headers := map[string]string{}
	bytes := make([]byte, 0, 100)
	for {
		str, err := rw.rw.ReadString('\n')
		if err != nil && str != "" {
			rw.rw.Read(bytes)
			return nil, jsonrpc2.NewError(jsonrpc2.ParseError, "Could not parse headers")
		}
		if err != nil && str == "" {
			rw.rw.Read(bytes)
            break
		}
		if strings.HasPrefix(str, "Content-Length:") && strings.HasSuffix(str, "\r\n") {
			segments := strings.Split(str, ":")
			value := strings.TrimLeft(segments[1], " ")
			value = strings.TrimRight(value, "\r\n")
			headers["Content-Length"] = value
		}
		if str == "\r\n" {
            break
		}
	}
	contentLength := headers["Content-Length"]
    length, err := strconv.ParseUint(contentLength, 10, 64)
    if err != nil || length == 0 {
		return nil, nil
	}
	bytes = make([]byte, length)
    if _, err := io.ReadFull(rw.rw.Reader, bytes); err != nil {
        return nil, jsonrpc2.NewError(jsonrpc2.ParseError, "Could not read message body")
    }

    jsonMessage, err := jsonrpc2.DecodeMessage(bytes)
	if err != nil {
        return nil, jsonrpc2.NewError(jsonrpc2.ParseError, fmt.Sprintf("Could not decode message body: %v", err))
	}

	return jsonMessage, nil
}
