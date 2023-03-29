package golog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	kratoslog "github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel/trace"
)

var _ kratoslog.Logger = (*stdLogger)(nil)

const (
	level       = "level"
	app         = "app"
	traceId     = "trace.id"
	spanId      = "span.id"
	uid         = "uid"
	pid         = "pid"
	threadName  = "threadName"
	callerStack = "callerStack"
)

// Logger is a logger interface.
type Logger interface {
	Log(level kratoslog.Level, keyvals ...interface{}) error
}

type ReportInterface interface {
	Report(msg string)
}

type Option func(logger *stdLogger)

type stdLogger struct {
	log  *log.Logger
	pool *sync.Pool

	level kratoslog.Level

	server string
	report func() ReportInterface
}

func WithFilterLevel(level kratoslog.Level) func(logger *stdLogger) {
	return func(logger *stdLogger) {
		logger.level = level
	}
}

func WithServerName(name string, version string) func(logger *stdLogger) {
	return func(logger *stdLogger) {
		logger.server = fmt.Sprintf("%s:%s", name, version)
	}
}

// NewFormatStdLogger new a logger with writer.
func NewFormatStdLogger(w io.Writer, opts ...Option) Logger {
	l := &stdLogger{
		log: log.New(w, "", 0),
		pool: &sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
		server: "-",
	}
	for _, o := range opts {
		o(l)
	}
	return kratoslog.With(l,
		app, l.server,
		traceId, traceID(),
		spanId, spanID(),
		uid, userid(),
		pid, gid(),
		threadName, "-",
		callerStack, caller(4),
	)
}

// Log print the kv pairs log.
func (l *stdLogger) Log(level kratoslog.Level, keyvals ...interface{}) error {

	if level < l.level {
		return nil
	}

	if len(keyvals) == 0 {
		return nil
	}
	if (len(keyvals) & 1) == 1 {
		keyvals = append(keyvals, "KEYVALS UNPAIRED")
	}
	runtime.Caller(4)
	buf := l.pool.Get().(*bytes.Buffer)
	buf.WriteString("[" + time.Now().Format("2006-01-02 15:04:05.000") + "] ")
	buf.WriteString(level.String())

	for i := 0; i < len(keyvals); i += 2 {
		switch keyvals[i] {
		case app:
			_, _ = fmt.Fprintf(buf, " [%s,", keyvals[i+1])
		case traceId:
			_, _ = fmt.Fprintf(buf, "%s,", keyvals[i+1])
		case spanId:
			_, _ = fmt.Fprintf(buf, "%s]", keyvals[i+1])
		case uid:
			_, _ = fmt.Fprintf(buf, " [uid=%s]", keyvals[i+1])
		case pid:
			//_, _ = fmt.Fprintf(buf, " %d", keyvals[i+1])
			_, _ = fmt.Fprintf(buf, " -1")
		case threadName:
			_, _ = fmt.Fprintf(buf, " [thread=%s]", keyvals[i+1])
		case callerStack:
			_, _ = fmt.Fprintf(buf, " [%s]:", keyvals[i+1])
		default:
			_, _ = fmt.Fprintf(buf, " %s=%v", keyvals[i], keyvals[i+1])
		}
	}
	_ = l.log.Output(4, buf.String()) //nolint:gomnd
	if level >= kratoslog.LevelError {
		rp := l.report()
		if rp != nil {
			rp.Report(buf.String())
		}
	}
	buf.Reset()
	l.pool.Put(buf)
	return nil
}

func (l *stdLogger) Close() error {
	return nil
}

func traceID() kratoslog.Valuer {
	return func(ctx context.Context) interface{} {
		if span := trace.SpanContextFromContext(ctx); span.HasTraceID() {
			return span.TraceID().String()
		}
		return "-"
	}
}

func spanID() kratoslog.Valuer {
	return func(ctx context.Context) interface{} {
		if span := trace.SpanContextFromContext(ctx); span.HasSpanID() {
			return span.SpanID().String()
		}
		return "-"
	}
}

func caller(depth int) kratoslog.Valuer {
	return func(context.Context) interface{} {
		_, file, line, _ := runtime.Caller(depth)
		idx := strings.LastIndexByte(file, '/')
		if idx == -1 {
			return file[idx+1:] + ":" + strconv.Itoa(line)
		}
		idx = strings.LastIndexByte(file[:idx], '/')
		return file[idx+1:] + ":" + strconv.Itoa(line)
	}
}

func gid() kratoslog.Valuer {
	return func(context.Context) interface{} {
		return os.Getpid()
	}
}

func userid() kratoslog.Valuer {
	return func(ctx context.Context) interface{} {
		v := ctx.Value(uid)
		if vv, ok := v.(string); ok {
			return vv
		}
		return ""
	}
}
