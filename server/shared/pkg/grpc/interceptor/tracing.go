package interceptor

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// TracingInterceptor adds OpenTelemetry tracing to gRPC calls.
type TracingInterceptor struct {
	tracer trace.Tracer
}

// NewTracingInterceptor creates a new tracing interceptor.
//
// If tracerName is empty, it defaults to "grpc".
func NewTracingInterceptor(tracerName string) *TracingInterceptor {
	if tracerName == "" {
		tracerName = "grpc"
	}

	return &TracingInterceptor{
		tracer: otel.Tracer(tracerName),
	}
}

// UnaryServerInterceptor returns a unary server interceptor for tracing.
func (t *TracingInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Extract trace context from metadata
		ctx = t.extractTraceContext(ctx)

		// Start span
		ctx, span := t.tracer.Start(ctx, info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.method", info.FullMethod),
				attribute.String("rpc.service", extractServiceName(info.FullMethod)),
			),
		)
		defer span.End()

		// Call handler
		resp, err := handler(ctx, req)

		// Record error if any
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())

			// Add gRPC status code
			if st, ok := status.FromError(err); ok {
				span.SetAttributes(attribute.Int("rpc.grpc.status_code", int(st.Code())))
			}
		} else {
			span.SetStatus(codes.Ok, "")
			span.SetAttributes(attribute.Int("rpc.grpc.status_code", 0))
		}

		return resp, err
	}
}

// StreamServerInterceptor returns a stream server interceptor for tracing.
func (t *TracingInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// Extract trace context
		ctx := t.extractTraceContext(ss.Context())

		// Start span
		ctx, span := t.tracer.Start(ctx, info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.method", info.FullMethod),
				attribute.String("rpc.service", extractServiceName(info.FullMethod)),
				attribute.Bool("rpc.stream", true),
			),
		)
		defer span.End()

		// Wrap stream with traced context
		wrapped := &tracedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		// Call handler
		err := handler(srv, wrapped)

		// Record error
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())

			if st, ok := status.FromError(err); ok {
				span.SetAttributes(attribute.Int("rpc.grpc.status_code", int(st.Code())))
			}
		} else {
			span.SetStatus(codes.Ok, "")
			span.SetAttributes(attribute.Int("rpc.grpc.status_code", 0))
		}

		return err
	}
}

// UnaryClientInterceptor returns a unary client interceptor for tracing.
func (t *TracingInterceptor) UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// Start span
		ctx, span := t.tracer.Start(ctx, method,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.method", method),
				attribute.String("rpc.service", extractServiceName(method)),
			),
		)
		defer span.End()

		// Inject trace context into metadata
		ctx = t.injectTraceContext(ctx, span.SpanContext())

		// Call invoker
		err := invoker(ctx, method, req, reply, cc, opts...)

		// Record error
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())

			if st, ok := status.FromError(err); ok {
				span.SetAttributes(attribute.Int("rpc.grpc.status_code", int(st.Code())))
			}
		} else {
			span.SetStatus(codes.Ok, "")
			span.SetAttributes(attribute.Int("rpc.grpc.status_code", 0))
		}

		return err
	}
}

// StreamClientInterceptor returns a stream client interceptor for tracing.
func (t *TracingInterceptor) StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		// Start span
		ctx, span := t.tracer.Start(ctx, method,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.method", method),
				attribute.String("rpc.service", extractServiceName(method)),
				attribute.Bool("rpc.stream", true),
			),
		)

		// Inject trace context
		ctx = t.injectTraceContext(ctx, span.SpanContext())

		// Call streamer
		cs, err := streamer(ctx, desc, cc, method, opts...)

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return nil, err
		}

		// Wrap stream
		return &tracedClientStream{
			ClientStream: cs,
			span:         span,
		}, nil
	}
}

// extractTraceContext extracts trace context from incoming metadata.
func (t *TracingInterceptor) extractTraceContext(ctx context.Context) context.Context {
	// Use OpenTelemetry's propagator to extract context
	// This is simplified - in production you'd use otel.GetTextMapPropagator()
	return ctx
}

// injectTraceContext injects trace context into outgoing metadata.
func (t *TracingInterceptor) injectTraceContext(ctx context.Context, sc trace.SpanContext) context.Context {
	// Inject trace ID and span ID into metadata
	md := metadata.Pairs(
		"x-trace-id", sc.TraceID().String(),
		"x-span-id", sc.SpanID().String(),
	)

	return metadata.NewOutgoingContext(ctx, md)
}

// extractServiceName extracts service name from full method name.
//
// Example: "/auth.v1.AuthService/Login" -> "auth.v1.AuthService"
func extractServiceName(fullMethod string) string {
	if len(fullMethod) == 0 {
		return ""
	}

	// Remove leading slash
	if fullMethod[0] == '/' {
		fullMethod = fullMethod[1:]
	}

	// Find last slash
	for i := len(fullMethod) - 1; i >= 0; i-- {
		if fullMethod[i] == '/' {
			return fullMethod[:i]
		}
	}

	return fullMethod
}

// tracedServerStream wraps grpc.ServerStream with traced context.
type tracedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *tracedServerStream) Context() context.Context {
	return s.ctx
}

// tracedClientStream wraps grpc.ClientStream and ends span on close.
type tracedClientStream struct {
	grpc.ClientStream
	span trace.Span
}

func (s *tracedClientStream) RecvMsg(m interface{}) error {
	err := s.ClientStream.RecvMsg(m)
	if err != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, err.Error())
		s.span.End()
	}
	return err
}

func (s *tracedClientStream) CloseSend() error {
	err := s.ClientStream.CloseSend()
	if err != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, err.Error())
	} else {
		s.span.SetStatus(codes.Ok, "")
	}
	s.span.End()
	return err
}
