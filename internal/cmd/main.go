package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
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

type application struct {
	config config
	logger *log.Logger
	models data.Models
	queue  chan struct{}
	jwtkey []byte
}

func main() {
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

	flag.StringVar(&cfg.db.host, "db-host", os.Getenv("DATABASE_HOST"), "PostgreSQL host")                 // Default to localhost for Docker Compose
	flag.StringVar(&cfg.db.name, "db-name", os.Getenv("DATABASE_NAME"), "PostgreSQL database name")        // Default to greenlight
	flag.StringVar(&cfg.db.user, "db-user", os.Getenv("DATABASE_USER"), "PostgreSQL user")                 // Default to user
	flag.StringVar(&cfg.db.password, "db-password", os.Getenv("DATABASE_PASSWORD"), "PostgreSQL password") // Default to password

	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", DbMaxOpenCons, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", DbMaxIdleCons, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", os.Getenv("DATABASE_MAX_IDLE_TIME"), "PostgreSQL max connection idle time")

	cfg.numWorkers = 50

	flag.Parse()

	logger.Printf("Config: %v", cfg)

	logger.Printf("DSN String: %s", cfg.db.dsn)

	db, err := openDB(cfg)
	if err != nil {
		logger.Fatal(err)
	}

	defer db.Close()

	logger.Printf("database connection established")

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
		jwtkey: []byte(jwtkey),
		queue:  make(chan struct{}, cfg.numWorkers),
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	logger.Printf("starting %s server on %s", cfg.env, srv.Addr)

	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall. SIGKILL but can"t be catch, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	// catching ctx.Done(). timeout of 5 seconds.
	select {
	case <-ctx.Done():
		log.Println("timeout of 5 seconds.")
	}
	log.Println("Server exiting")

	err = srv.ListenAndServe()
	logger.Fatal(err)

}

func openDB(cfg config) (*sql.DB, error) {
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
