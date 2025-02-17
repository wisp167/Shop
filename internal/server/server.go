package server

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/wisp167/Shop/internal/data"
)

const version = "1.0.0"

type config struct {
	port       int
	env        string
	numWorkers int
	db         struct {
		dsn          string
		host         string
		name         string
		user         string
		password     string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
	}
}

type Application struct {
	config config
	logger *log.Logger
	models data.Models
	queue  chan struct{}
	jwtkey []byte
	server *http.Server
}

func SetupApplication() (*Application, error) {
	var cfg config
	var jwtkey string

	godotenv.Load(".env")

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

	EnvPort, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse PORT: %v", err)
	}

	DbMaxOpenCons, err := strconv.Atoi(os.Getenv("DATABASE_MAX_OPEN_CONNS"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse DATABASE_MAX_OPEN_CONNS: %v", err)
	}
	DbMaxIdleCons, err := strconv.Atoi(os.Getenv("DATABASE_MAX_IDLE_CONNS"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse DATABASE_MAX_IDLE_CONNS: %v", err)
	}
	jwtKey := os.Getenv("JWT_KEY")
	if jwtKey == "" {
		return nil, fmt.Errorf("JWT_KEY environment variable is required")
	}
	flag.IntVar(&cfg.port, "port", EnvPort, "API server port")
	flag.StringVar(&cfg.env, "env", os.Getenv("ENV"), "Environment (development|staging|production)")

	flag.StringVar(&cfg.db.host, "db-host", os.Getenv("DATABASE_HOST"), "PostgreSQL host")
	flag.StringVar(&cfg.db.name, "db-name", os.Getenv("DATABASE_NAME"), "PostgreSQL database name")
	flag.StringVar(&cfg.db.user, "db-user", os.Getenv("DATABASE_USER"), "PostgreSQL user")
	flag.StringVar(&cfg.db.password, "db-password", os.Getenv("DATABASE_PASSWORD"), "PostgreSQL password")

	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", DbMaxOpenCons, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", DbMaxIdleCons, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", os.Getenv("DATABASE_MAX_IDLE_TIME"), "PostgreSQL max connection idle time")

	cfg.numWorkers = 50

	flag.Parse()

	logger.Printf("Config: %v", cfg)

	// Open the database connection
	db, err := OpenDB(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	app := &Application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
		jwtkey: []byte(jwtkey),
		queue:  make(chan struct{}, cfg.numWorkers),
	}

	return app, nil
}
func (app *Application) Start() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	app.server = srv

	app.logger.Printf("starting %s server on %s", app.config.env, srv.Addr)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.logger.Fatalf("listen: %s\n", err)
		}
	}()

	return nil
}

func (app *Application) Stop() error {
	if app.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %v", err)
	}

	app.logger.Println("server stopped")
	return nil
}

/*
func StartServer() {
	cfg, app := SetupApplication()
	defer app.server.Shutdown(context.Background())
	logger := app.logger

	db, err := OpenDB(cfg)
	if err != nil {
		logger.Fatal(err)
	}
	defer db.Close()

	logger.Printf("database connection established")

	app.models = data.NewModels(db)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	app.server = srv

	logger.Printf("starting %s server on %s", cfg.env, srv.Addr)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	select {
	case <-ctx.Done():
		log.Println("timeout of 5 seconds.")
	}
	log.Println("Server exiting")
}

func SetupApplication() (config, *Application) {
	var cfg config
	var jwtkey string

	godotenv.Load(".env")

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

	EnvPort, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		logger.Fatal(err)
	}

	DbMaxOpenCons, err := strconv.Atoi(os.Getenv("DATABASE_MAX_OPEN_CONNS"))
	if err != nil {
		logger.Fatal(err)
	}
	DbMaxIdleCons, err := strconv.Atoi(os.Getenv("DATABASE_MAX_IDLE_CONNS"))
	if err != nil {
		logger.Fatal(err)
	}

	flag.StringVar(&jwtkey, "jwt-key", os.Getenv("JWT_KEY"), "jwt key")
	flag.IntVar(&cfg.port, "port", EnvPort, "API server port")
	flag.StringVar(&cfg.env, "env", os.Getenv("ENV"), "Enviroment (development|staging|production)")

	flag.StringVar(&cfg.db.host, "db-host", os.Getenv("DATABASE_HOST"), "PostgreSQL host")
	flag.StringVar(&cfg.db.name, "db-name", os.Getenv("DATABASE_NAME"), "PostgreSQL database name")
	flag.StringVar(&cfg.db.user, "db-user", os.Getenv("DATABASE_USER"), "PostgreSQL user")
	flag.StringVar(&cfg.db.password, "db-password", os.Getenv("DATABASE_PASSWORD"), "PostgreSQL password")

	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", DbMaxOpenCons, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", DbMaxIdleCons, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", os.Getenv("DATABASE_MAX_IDLE_TIME"), "PostgreSQL max connection idle time")

	cfg.numWorkers = 50

	flag.Parse()

	logger.Printf("Config: %v", cfg)

	logger.Printf("DSN String: %s", cfg.db.dsn)

	app := &Application{
		config: cfg,
		logger: logger,
		jwtkey: []byte(jwtkey),
		queue:  make(chan struct{}, cfg.numWorkers),
	}

	return cfg, app
}
*/

func OpenDB(cfg config) (*sql.DB, error) {
	cfg.db.dsn = fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=disable",
		cfg.db.user,
		cfg.db.password,
		cfg.db.host,
		cfg.db.name,
	)
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	db.SetMaxIdleConns(cfg.db.maxIdleConns)

	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxIdleTime(duration)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}
	return db, nil
}
