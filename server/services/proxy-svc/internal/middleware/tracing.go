package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	tracerName = "proxy-svc"
)

// Tracing OpenTelemetry链路追踪中间件
func Tracing(serviceName string) gin.HandlerFunc {
	tracer := otel.Tracer(tracerName)
	propagator := otel.GetTextMapPropagator()

	return func(c *gin.Context) {
		// 从HTTP头提取trace上下文
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// 开始新的Span
		spanName := c.Request.Method + " " + c.FullPath()
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.method", c.Request.Method),
				attribute.String("http.url", c.Request.URL.String()),
				attribute.String("http.scheme", c.Request.URL.Scheme),
				attribute.String("http.host", c.Request.Host),
				attribute.String("http.user_agent", c.Request.UserAgent()),
				attribute.String("http.client_ip", c.ClientIP()),
				attribute.String("service.name", serviceName),
			),
		)
		defer span.End()

		// 将trace上下文注入到gin.Context
		c.Request = c.Request.WithContext(ctx)

		// 注入TraceID到响应头（便于日志关联）
		if span.SpanContext().HasTraceID() {
			c.Header("X-Trace-ID", span.SpanContext().TraceID().String())
		}

		// 处理请求
		c.Next()

		// 记录响应状态码
		statusCode := c.Writer.Status()
		span.SetAttributes(
			attribute.Int("http.status_code", statusCode),
			attribute.Int("http.response_size", c.Writer.Size()),
		)

		// 记录错误
		if statusCode >= 400 {
			span.SetStatus(codes.Error, "HTTP error")
			if len(c.Errors) > 0 {
				span.RecordError(c.Errors.Last())
				span.SetAttributes(attribute.String("error.message", c.Errors.Last().Error()))
			}
		} else {
			span.SetStatus(codes.Ok, "")
		}
	}
}

// GetTraceID 从context中获取TraceID（用于日志关联）
func GetTraceID(c *gin.Context) string {
	span := trace.SpanFromContext(c.Request.Context())
	if span.SpanContext().HasTraceID() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID 从context中获取SpanID
func GetSpanID(c *gin.Context) string {
	span := trace.SpanFromContext(c.Request.Context())
	if span.SpanContext().HasSpanID() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}
