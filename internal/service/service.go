package service

import (
	"context"
	"errors"
	"os"
	"sync"

	"github.com/macleodmac/pglet/internal/client"
	"github.com/macleodmac/pglet/internal/repository"
)

var (
	ErrNotConnected  = errors.New("not connected to database")
	ErrNoAPIKey      = errors.New("AI API key not configured, set ANTHROPIC_API_KEY environment variable")
	ErrNoTransaction = errors.New("no open transaction for this tab")
	ErrTxOpen        = errors.New("commit or roll back open transactions before changing connection options")
)

// Service holds all shared state and provides business logic methods.
// Methods are namespaced by the file they live in (connection.go, schema.go, etc.).
type Service struct {
	mu         sync.RWMutex
	client     *client.Client
	connecting bool
	Repo       *repository.Repository
	Version    string

	queryMu sync.Mutex
	running map[string]context.CancelFunc

	txMu sync.Mutex
	txns map[string]*client.Tx
}

func New(repo *repository.Repository, version string) *Service {
	return &Service{
		Repo:    repo,
		Version: version,
		running: make(map[string]context.CancelFunc),
		txns:    make(map[string]*client.Tx),
	}
}

func (s *Service) SetClient(cl *client.Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.client = cl
	s.connecting = false
}

func (s *Service) SetConnecting(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connecting = v
}

func (s *Service) IsConnecting() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connecting
}

func (s *Service) GetClient() *client.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client
}

func (s *Service) SwapClient(cl *client.Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client != nil {
		s.client.Close()
	}
	s.client = cl
}

// requireClient returns the current client or ErrNotConnected.
func (s *Service) requireClient() (*client.Client, error) {
	cl := s.GetClient()
	if cl == nil {
		return nil, ErrNotConnected
	}
	return cl, nil
}

// AppInfo returns the app version and whether AI is enabled.
func (s *Service) AppInfo() (version string, aiEnabled bool) {
	return s.Version, os.Getenv("ANTHROPIC_API_KEY") != ""
}

// QueryError wraps a query execution error. The handler uses this to return
// the error inside a 200 OK result (matching the existing API contract).
type QueryError struct {
	Err error
}

func (e *QueryError) Error() string { return e.Err.Error() }
func (e *QueryError) Unwrap() error { return e.Err }
