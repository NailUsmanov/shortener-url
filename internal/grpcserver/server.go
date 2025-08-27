package grpcserver

import (
	"fmt"
	"net"
	"strings"

	"context"

	shortenerpb "github.com/NailUsmanov/practicum-shortener-url/internal/genproto/shortener/v1"
	"github.com/NailUsmanov/practicum-shortener-url/internal/storage"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Минимальный конфиг gRPC-слоя. Зеркалит нужные опции из HTTP.
type Config struct {
	Addr          string // ":3200" — где слушать gRPC
	BaseURL       string
	TrustedSubnet string // CIDR, чтобы ограничить доступ к Stats
}

// Server Тонкий фасад gRPC над существующим storage.Storage и бизнес-логикой.
type Server struct {
	shortenerpb.UnimplementedShortenerServiceServer
	log        *zap.SugaredLogger
	st         storage.Storage
	cfg        Config
	allowStats func(ctx context.Context) error // сюда положим проверку trusted subnet
}

func New(log *zap.SugaredLogger, st storage.Storage, cfg Config) *Server {
	return &Server{
		log:        log,
		st:         st,
		cfg:        cfg,
		allowStats: CheckTrustedSubnet(cfg.TrustedSubnet), // функция из interceptors.go
	}
}

// Вспомогательная сборка полного короткого URL из base + id.
func (s *Server) ShortURL(id string) string {
	return fmt.Sprintf("%s/%s", s.cfg.BaseURL, id)
}

// Serve поднимает gRPC-сервер, регистрирует сервис и начинает слушать addr.
// Возвращает *grpc.Server, чтобы ты мог сделать graceful shutdown через GracefulStop().
func (s *Server) Serve(addr string) (*grpc.Server, error) {
	gs := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			UnaryRecovery(s.log),
			UnaryLogging(s.log),
			UnaryAuth(),
		),
	)

	shortenerpb.RegisterShortenerServiceServer(gs, s)
	reflection.Register(gs)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	s.log.Infof("gRPC listening on %s", addr)
	go func() { _ = gs.Serve(l) }()
	return gs, nil
}

// Shorten сокращает оригинальный URL и возвращает короткий URL.
func (s *Server) Shorten(ctx context.Context, req *shortenerpb.ShortenRequest) (*shortenerpb.ShortenResponse, error) {
	userID, ok := UnaryUserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, status.Error(codes.Unauthenticated, "no user")
	}
	if req.GetOriginalUrl() == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid url")
	}
	if key, err := s.st.GetByURL(ctx, req.GetOriginalUrl(), userID); err == nil && key != "" {
		return &shortenerpb.ShortenResponse{
			ShortUrl:      s.ShortURL(key),
			AlreadyExists: true,
		}, nil
	}
	key, err := s.st.Save(ctx, req.GetOriginalUrl(), userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "save failed")
	}

	return &shortenerpb.ShortenResponse{ShortUrl: s.ShortURL(key)}, nil
}

// ShortenBatch позволяет отправить запрос из нескольких ссылок для сокращения и возвращает массив коротких URL.
func (s *Server) ShortenBatch(ctx context.Context, req *shortenerpb.ShortenBatchRequest) (*shortenerpb.ShortenBatchResponse, error) {
	userID, ok := UnaryUserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, status.Error(codes.Unauthenticated, "no user")
	}
	items := req.GetItems()
	if len(items) == 0 {
		return nil, status.Error(codes.InvalidArgument, "empty batch")
	}

	urls := make([]string, 0, len(items))
	for _, it := range items {
		orig := strings.TrimSpace(it.GetOriginalUrl())
		if orig == "" {
			return nil, status.Error(codes.InvalidArgument, "invalid url")
		}
		urls = append(urls, orig)
	}
	keys, err := s.st.SaveInBatch(ctx, urls, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "batch save failed")
	}
	if len(keys) != len(items) {
		return nil, status.Error(codes.Internal, "batch size mismatch")
	}
	out := make([]*shortenerpb.ShortenBatchResponseItem, 0, len(items))
	for i, it := range items {
		out = append(out, &shortenerpb.ShortenBatchResponseItem{
			CorrelationId: it.GetCorrelationId(),
			ShortUrl:      s.ShortURL(keys[i]),
		})
	}
	return &shortenerpb.ShortenBatchResponse{Items: out}, nil
}

// Redirect перенаправляет запрос с короткой ссылки на оригинальную.
func (s *Server) Redirect(ctx context.Context, req *shortenerpb.RedirectRequest) (*shortenerpb.RedirectResponse, error) {

	shortID := strings.TrimSpace(req.GetShortId())
	if shortID == "" {
		return nil, status.Error(codes.InvalidArgument, "empty short_id")
	}

	orig, err := s.st.Get(ctx, shortID)
	if err != nil {
		switch err {
		case storage.ErrNotFound:
			return nil, status.Error(codes.NotFound, "not found")
		case storage.ErrDeleted:
			return nil, status.Error(codes.FailedPrecondition, "url deleted")
		default:
			return nil, status.Error(codes.Internal, "get original url failed")
		}
	}
	return &shortenerpb.RedirectResponse{OriginalUrl: orig}, nil
}

// ListUserURLs возвращает список пар короткого и оригинального URL для пользователя.
func (s *Server) ListUserURLs(ctx context.Context, _ *emptypb.Empty) (*shortenerpb.ListUserURLsResponse, error) {
	userID, ok := UnaryUserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, status.Error(codes.Unauthenticated, "no user")
	}

	urls, err := s.st.GetUserURLS(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "get urls failed")
	}
	if len(urls) == 0 {
		return &shortenerpb.ListUserURLsResponse{}, nil
	}

	items := make([]*shortenerpb.ListUserURLsResponseItem, 0, len(urls))
	for id, orig := range urls {
		items = append(items, &shortenerpb.ListUserURLsResponseItem{
			ShortUrl:    s.ShortURL(id),
			OriginalUrl: orig,
		})
	}
	return &shortenerpb.ListUserURLsResponse{Items: items}, nil
}

// DeleteUserURLs удаляет переданные URL из базы для конкретного пользователя.
func (s *Server) DeleteUserURLs(ctx context.Context, req *shortenerpb.DeleteUserURLsRequest) (*emptypb.Empty, error) {
	userID, ok := UnaryUserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, status.Error(codes.Unauthenticated, "no user")
	}
	urls := req.GetShortUrls()
	if len(urls) == 0 {
		return nil, status.Error(codes.InvalidArgument, "empty items")
	}
	items := make([]string, 0, len(urls))

	for _, i := range urls {
		items = append(items, trimID(i))
	}

	if err := s.st.MarkAsDeleted(ctx, items, userID); err != nil {
		return nil, status.Error(codes.Internal, "deleted failed")
	}
	return &emptypb.Empty{}, nil
}

// trimID извлекает идентификатор из полного short_url, если пришел URL, а не id.
func trimID(shortURL string) string {
	i := len(shortURL) - 1
	for i >= 0 && shortURL[i] != '/' {
		i--
	}
	if i >= 0 && i+1 < len(shortURL) {
		return shortURL[i+1:]
	}
	return shortURL
}

// Ping проверяет доступность стораджа.
func (s *Server) Ping(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.st.Ping(ctx); err != nil {
		return nil, status.Error(codes.Internal, "db not avaible")
	}
	return &emptypb.Empty{}, nil
}

// Stats возвращает агрегированную статистику. Доступ ограничен trusted subnet.
func (s *Server) Stats(ctx context.Context, _ *emptypb.Empty) (*shortenerpb.StatsResponse, error) {
	if s.allowStats != nil {
		if err := s.allowStats(ctx); err != nil {
			return nil, err
		}
	}
	users, err := s.st.CountUsers(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "count users failed")
	}

	urls, err := s.st.CountURL(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "count urls failed")
	}

	return &shortenerpb.StatsResponse{Users: int32(users), Urls: int32(urls)}, nil
}
