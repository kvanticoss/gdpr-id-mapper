package svc

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

const (
	defaultTerminationGracePeriod = 15 * time.Second
)

type SVC struct {
	Name    string
	Version string

	Router *http.ServeMux

	TerminationGracePeriod time.Duration
	signals                chan os.Signal

	logger             *zap.Logger
	stdLogger          *log.Logger
	atom               zap.AtomicLevel
	loggerRedirectUndo func()

	workers            map[string]Worker
	workersAdded       []string
	workersInitialized []string
}

// New instantiates a new service by parsing configuration and initializing a
// logger.
func New(name, version string, opts ...Option) (*SVC, error) {
	s := &SVC{
		Name:    name,
		Version: version,

		Router: http.NewServeMux(),

		TerminationGracePeriod: defaultTerminationGracePeriod,
		signals:                make(chan os.Signal, 3),

		workers:            map[string]Worker{},
		workersAdded:       []string{},
		workersInitialized: []string{},
	}

	if err := WithDevelopmentLogger()(s); err != nil {
		return nil, err
	}

	// Apply options
	for _, o := range opts {
		if err := o(s); err != nil {
			return nil, err
		}
	}

	return s, nil
}

// AddWorker adds a named worker to the service. Added workers order is
// maintained.
func (s *SVC) AddWorker(name string, w Worker) {
	if _, exists := s.workers[name]; exists {
		s.logger.Fatal("Duplicate worker names!", zap.String("name", name), zap.Stack("stacktrace"))
	}
	if _, ok := w.(Healther); !ok {
		s.logger.Warn("Worker does not implement Healther interface", zap.String("worker", name))
	}
	// Track workers as ordered set to initialize them in order.
	s.workersAdded = append(s.workersAdded, name)
	s.workers[name] = w
}

// Run runs the service until either receiving an interrupt or a worker
// terminates.
func (s *SVC) Run() {
	s.logger.Info("Starting up service")

	defer func() {
		s.logger.Info("Shutting down service", zap.Duration("termination_grace_period", s.TerminationGracePeriod))
		s.terminateWorkers()
		s.logger.Info("Service shutdown completed")
		_ = s.logger.Sync()
		s.loggerRedirectUndo()
	}()

	// Initializing workers in added order.
	for _, name := range s.workersAdded {
		s.logger.Debug("Initializing worker", zap.String("worker", name))
		w := s.workers[name]
		if err := w.Init(s.logger.Named(name)); err != nil {
			s.logger.Error("Could not initialize service", zap.String("worker", name), zap.Error(err))
			return
		}
		s.workersInitialized = append(s.workersInitialized, name)
	}

	errs := make(chan error)
	wg := sync.WaitGroup{}
	for _, w := range s.workers {
		wg.Add(1)
		go func(w Worker) {
			defer recoverWait(&wg, errs)
			if err := w.Run(); err != nil {
				errs <- err
			}
		}(w)
	}

	signal.Notify(s.signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	select {
	case err := <-errs:
		s.logger.Fatal("Worker Init/Run failure", zap.Error(err), zap.Stack("stacktrace"))
	case sig := <-s.signals:
		s.logger.Warn("Caught signal", zap.String("signal", sig.String()))
	case <-waitGroupToChan(&wg):
		s.logger.Info("All workers have finished")
	}
}

// Shutdown signals the framework to terminate any already started workers and
// shutdown the service.
// The call is non-blocking. Terminating the workers comes with the guarantees
// as the `Run` method: All workers are given a total terminate grace-period
// until the service goes ahead completes the shutdown phase.
func (s *SVC) Shutdown() {
	s.signals <- syscall.SIGTERM
}

// MustInit is a convenience function to check for and halt on errors.
func MustInit(s *SVC, err error) {
	if err != nil {
		if s == nil || s.logger == nil {
			panic(err)
		}
		s.logger.Fatal("Service initialization failed", zap.Error(err), zap.Stack("stacktrace"))
	}
}

// Logger returns the service's logger. Logger might be nil if New() fails.
func (s *SVC) Logger() *zap.Logger {
	return s.logger
}

func (s *SVC) terminateWorkers() {
	s.logger.Info("Terminating workers down service", zap.Duration("termination_grace_period", s.TerminationGracePeriod))

	// terminate only initialized workers
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, name := range s.workersInitialized {
			defer func(name string) {
				w := s.workers[name]
				if err := w.Terminate(); err != nil {
					s.logger.Error("Terminated with error",
						zap.String("worker", name),
						zap.Error(err))
				}
				s.logger.Info("Worker terminated", zap.String("worker", name))
			}(name)
		}
	}()
	waitGroupTimeout(&wg, s.TerminationGracePeriod)
	s.logger.Info("All workers terminated")
}

func waitGroupTimeout(wg *sync.WaitGroup, d time.Duration) {
	select {
	case <-waitGroupToChan(wg):
	case <-time.After(d):
	}
}

func waitGroupToChan(wg *sync.WaitGroup) <-chan struct{} {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	return c
}

func recoverWait(wg *sync.WaitGroup, errors chan<- error) {
	wg.Done()
	if r := recover(); r != nil {
		if err, ok := r.(error); ok {
			errors <- err
		} else {
			errors <- fmt.Errorf("%v", r)
		}
	}
}
