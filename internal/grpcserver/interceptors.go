package grpcserver

import (
	"net"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"google.golang.org/grpc/status"
)

// ctxUserIDKey — тип-ключ для userID в контексте (чтобы избежать коллизий).
type ctxUserIDKey struct{}

// UnaryAuth — унарный перехватчик аутентификации.
// Требует наличие metadata "user-id: <string>" и кладёт его в контекст.
func UnaryAuth() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {

		if info.FullMethod == "/shortener.v1.ShortenerService/Redirect" ||
			info.FullMethod == "/shortener.v1.ShortenerService/Ping" {
			return handler(ctx, req)
		}
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "no metadata")
		}

		values := md.Get("user-id")
		if len(values) == 0 || strings.TrimSpace(values[0]) == "" {
			return nil, status.Error(codes.Unauthenticated, "missing user-id")
		}

		ctx = context.WithValue(ctx, ctxUserIDKey{}, values[0])
		return handler(ctx, req)
	}
}

// UnaryUserIDFromContext достает userID, который положил UnaryAuth.
func UnaryUserIDFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(ctxUserIDKey{})

	if v == nil {
		return "", false
	}

	id, ok := v.(string)
	return id, ok

}

// UnaryLogging - логирование всех унарных RPC: метод, код, длительность.
func UnaryLogging(log *zap.SugaredLogger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		code := status.Code(err)
		log.Infow("grpc",
			"method", info.FullMethod,
			"code", code.String(),
			"dur_ms", time.Since(start).Milliseconds())

		return resp, err
	}
}

// UnaryRecovery - ловит паники и переводит их в codes.Internal.
func UnaryRecovery(log *zap.SugaredLogger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("panic in %s: %v", info.FullMethod, r)
				err = status.Error(codes.Internal, "internal error")
			}
		}()
		return handler(ctx, req)
	}
}

// CheckTrustedSubnet возвращает функцию-проверку IP клиента для метода Stats.
func CheckTrustedSubnet(trustedCIDR string) func(ctx context.Context) error {
	cidr := strings.TrimSpace(trustedCIDR)
	if cidr == "" {
		return func(ctx context.Context) error {
			return status.Error(codes.PermissionDenied, "trusted subnet not configuret")
		}
	}

	_, nw, err := net.ParseCIDR(cidr)
	if err != nil {
		return func(ctx context.Context) error {
			return status.Error(codes.PermissionDenied, "bad trusted subnet")
		}
	}
	return func(ctx context.Context) error {
		p, ok := peer.FromContext(ctx)
		if !ok || p.Addr == nil {
			return status.Error(codes.PermissionDenied, "no peer addr")
		}
		host, _, _ := net.SplitHostPort(p.Addr.String())
		ip := net.ParseIP(host)

		if ip == nil || !nw.Contains(ip) {
			return status.Error(codes.PermissionDenied, "fobidden by trusted subnet")
		}
		return nil
	}
}

// Chain возвращает СерверОптион с цепочкой перехватчиков. Подключаем в grpc.NewServer.
func Chain(log *zap.SugaredLogger) grpc.ServerOption {
	return grpc.ChainUnaryInterceptor(
		UnaryRecovery(log),
		UnaryLogging(log),
		UnaryAuth(),
	)
}
