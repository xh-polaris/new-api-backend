package model

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"one-api/logger"
	"time"
)

type APILog struct {
	Start time.Time `json:"start" gorm:"column:start;type:datetime;not null"`
	End   time.Time `json:"end" gorm:"column:end;type:datetime;not null"`
	Req   string    `json:"req" gorm:"column:req;type:text;charset=utf8mb4;collate=utf8mb4_0900_ai_ci"`
	Res   string    `json:"res" gorm:"column:res;type:text;charset=utf8mb4;collate=utf8mb4_0900_ai_ci"`
	Err   string    `json:"err,omitempty" gorm:"column:err;type:varchar(255);charset=utf8mb4;collate=utf8mb4_0900_ai_ci"`
}

// LoggingTransport 是一个自定义 Transport，用于打印 HTTP 请求和响应
type LoggingTransport struct {
	Transport http.RoundTripper // 底层 Transport（默认使用 http.DefaultTransport）
}

func NewLoggingTransport() *LoggingTransport {
	return &LoggingTransport{
		Transport: http.DefaultTransport,
	}
}

// RoundTrip 实现 http.RoundTripper 接口，拦截请求和响应
func (t *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Path != "/v1/chat/completions" { // 如果不是目标路径，直接转发请求
		return t.Transport.RoundTrip(req)
	}

	l := &APILog{Start: time.Now()}
	// 请求
	if req.Body != nil {
		reqBody, err := io.ReadAll(req.Body)
		req.Body, l.Req = io.NopCloser(bytes.NewBuffer(reqBody)), string(reqBody)
		if err != nil {
			l.Err = err.Error()
			PostProcess(l)
			return nil, err
		}
	}

	// 响应
	resp, err := t.Transport.RoundTrip(req)
	if err != nil {
		l.Err = err.Error()
		PostProcess(l)
		return nil, err
	}

	// 使用TeeReader复制一份响应, 当返回结束时buf会得到拷贝的响应
	var buf bytes.Buffer
	r := io.TeeReader(resp.Body, &buf)
	resp.Body = &rc{Reader: r, closeFunc: func() error {
		l.Res = buf.String()
		PostProcess(l)
		return nil
	}}
	return resp, nil
}

type rc struct {
	io.Reader
	closeFunc func() error
	closed    bool
}

func (rc *rc) Close() error {
	if rc.closeFunc != nil && !rc.closed {
		rc.closed = true
		return rc.closeFunc()
	}
	return nil
}

func PostProcess(l *APILog) {
	go func() {
		l.End = time.Now()
		if err := DB.Table("api_log").Create(l).Error; err != nil {
			logger.LogError(context.Background(), fmt.Sprintf("err: %s \n%+v", err, l))
		}
	}()
}
