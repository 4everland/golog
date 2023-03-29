package golog

import (
	"bytes"
	"context"
	"fmt"
	kratoslog "github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel/sdk/trace"
	"io"
	"regexp"
	"testing"
)

const (
	exampleLog  = "[2022-11-15 11:02:00.093] INFO [app:version,traceId,spanId] [uid=uid] 123 [thread=threadName] [callerStack]: message"
	exampleLog2 = "[2022-11-15 11:02:00.093] INFO [app:version,,] [uid=uid] 123 [thread=threadName] [callerStack]: message"
)

var (
	parseRegex = regexp.MustCompile(`\[[\d|\s|\\.|\-|:]+]\s+(\w+)\s+\[([^,]*),([^,]*),([^]]*)]\s+\[uid=([^]]*)]\s+(\-?\d*)\s+\[thread=([^]]*)]\s+\[([^]]+)]:\s+(.*)`)
)

// parse params:
// level,app, traceId, spanId,uid, pid, threadName, callerStack, message

func TestRegex(t *testing.T) {
	if err := matchRegex(exampleLog); err != nil {
		t.Fatalf(err.Error())
	}
	if err := matchRegex(exampleLog2); err != nil {
		t.Fatalf(err.Error())
	}
}

func matchRegex(logStr string) error {
	res := parseRegex.FindStringSubmatch(logStr)
	if len(res) != 10 {
		return fmt.Errorf("parse log error: %v", res)
	}
	fmt.Printf("level: %s,\t", res[1])
	fmt.Printf("app: %s,\t", res[2])
	fmt.Printf("traceId: %s,\t", res[3])
	fmt.Printf("spanId: %s,\t", res[4])
	fmt.Printf("uid: %s,\t", res[5])
	fmt.Printf("pid: %s,\t", res[6])
	fmt.Printf("threadName: %s,\t", res[7])
	fmt.Printf("callerStack: %s,\t", res[8])
	fmt.Printf("message: %s\n", res[9])

	return nil
}

func TestLog(t *testing.T) {
	var data = make([]byte, 4096)
	buf := bytes.NewBuffer(data)
	logger := NewFormatStdLogger(buf, WithServerName("test", "0.0.1"), WithFilterLevel(kratoslog.LevelInfo))
	ll := kratoslog.NewHelper(logger)
	ll.Infof("test log %s", "aaa")
	d, _ := io.ReadAll(buf)
	if err := matchRegex(string(d)); err != nil {
		t.Fatalf("%s, err:%s", string(d), err.Error())
	}
	ll.Warnf("test warn log %s", "aaa")
	d, _ = io.ReadAll(buf)
	if err := matchRegex(string(d)); err != nil {
		t.Fatalf("%s, err:%s", string(d), err.Error())
	}
	p := trace.NewTracerProvider()
	tt := p.Tracer("aaa")
	ctx, sp := tt.Start(context.WithValue(context.Background(), "uid", "-"), "span")
	defer sp.End()

	ll.WithContext(ctx).Warnf("test trace log %s", "aaa")
	d, _ = io.ReadAll(buf)
	if err := matchRegex(string(d)); err != nil {
		t.Fatalf("%s, err:%s", string(d), err.Error())
	}
}
