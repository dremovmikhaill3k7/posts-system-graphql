package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/vektah/gqlparser/v2/ast"

	"posts_service/internal/auth"
	"posts_service/internal/config"
	"posts_service/internal/graph"
	"posts_service/internal/loaders"
	"posts_service/internal/pubsub"
	"posts_service/internal/repository"
	"posts_service/internal/storage/memory"
	"posts_service/internal/storage/postgres"
)

const defaultPort = ":8080"

type App struct {
	httpServer *http.Server
	db         *sql.DB
}

func New(ctx context.Context) (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки конфигурации: %w", err)
	}

	var repo repository.Repository
	var db *sql.DB

	switch cfg.StorageMode {
	case config.StorageMemory:
		repo = memory.NewRepository()
		log.Println("режим хранения: in-memory")
	case config.StoragePostgres:
		u := url.URL{
			Scheme:   "postgres",
			User:     url.UserPassword(cfg.POSTGRES_USER, cfg.POSTGRES_PASSWORD),
			Host:     fmt.Sprintf("%s:%s", cfg.POSTGRES_HOST, cfg.POSTGRES_PORT),
			Path:     cfg.POSTGRES_DB,
			RawQuery: "sslmode=disable",
		}
		db, err = sql.Open("postgres", u.String())
		if err != nil {
			return nil, fmt.Errorf("ошибка DSN: %w", err)
		}
		if err = db.PingContext(ctx); err != nil {
			return nil, fmt.Errorf("ошибка подключения к БД: %w", err)
		}
		repo = postgres.NewRepository(db)
		log.Println("режим хранения: postgres")
	}

	port := cfg.PORT
	if port == "" {
		port = defaultPort
	}
	if port[0] != ':' {
		port = ":" + port
	}

	jwtSvc, err := auth.NewJWT(cfg.JWTSecret, cfg.JWTExpiration, cfg.CookieSecure, cfg.CookieDomain)
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации JWT: %w", err)
	}

	ps := pubsub.NewCommentPubSub()
	resolver := graph.NewResolver(repo, ps, jwtSvc)

	srv := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
	})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.Use(extension.Introspection{})
	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	r := chi.NewRouter()
	r.Use(auth.Middleware(jwtSvc))
	r.Handle("/", playground.Handler("GraphQL playground", "/v1/query"))
	r.Handle("/v1/query", loaders.Middleware(repo, srv))
	r.HandleFunc("/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	httpServer := &http.Server{
		Addr:    port,
		Handler: r,
	}

	return &App{
		httpServer: httpServer,
		db:         db,
	}, nil
}

func (a *App) Run(ctx context.Context) {
	if a.db != nil {
		defer a.db.Close()
	}

	go func() {
		log.Printf("Сервер запущен на %s", a.httpServer.Addr)
		if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("ошибка сервера: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("ошибка остановки сервера: %v", err)
	}

	log.Println("Сервер успешно остановлен")
}
