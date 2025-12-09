package health

import (
	"context"
	"sync"
	"time"
)

type HealthChecker interface {
	Check(ctx context.Context) error
	Name() string
}

type HealthService struct {
	checkers     map[string]HealthChecker
	mu           sync.RWMutex
	ready        bool
	shuttingDown bool
}

func NewHealthService() *HealthService {
	return &HealthService{
		checkers: make(map[string]HealthChecker),
	}
}

func (h *HealthService) RegisterChecker(checker HealthChecker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkers[checker.Name()] = checker
}

func (h *HealthService) SetReady(ready bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ready = ready
}

func (h *HealthService) SetShuttingDown(shuttingDown bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.shuttingDown = shuttingDown
}

func (h *HealthService) Check(ctx context.Context) map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make(map[string]interface{})
	result["status"] = "healthy"
	result["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	result["ready"] = h.ready
	result["shutting_down"] = h.shuttingDown

	checkResults := make(map[string]interface{})
	for name, checker := range h.checkers {
		if err := checker.Check(ctx); err != nil {
			checkResults[name] = map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			}
			result["status"] = "unhealthy"
		} else {
			checkResults[name] = map[string]interface{}{
				"status": "healthy",
			}
		}
	}
	result["checks"] = checkResults

	return result
}

type CertificateChecker struct {
	certFile string
}

func NewCertificateChecker(certFile string) *CertificateChecker {
	return &CertificateChecker{certFile: certFile}
}

func (c *CertificateChecker) Name() string {
	return "certificate"
}

func (c *CertificateChecker) Check(ctx context.Context) error {
	//  TODO: Implement certificate expiry check
	return nil
}

type TunnelConnectionChecker struct {
	minConnections int
}

func NewTunnelConnectionChecker(minConnections int) *TunnelConnectionChecker {
	return &TunnelConnectionChecker{minConnections: minConnections}
}

func (t *TunnelConnectionChecker) Name() string {
	return "tunnel_connections"
}

func (t *TunnelConnectionChecker) Check(ctx context.Context) error {
	// Implement connection count check
	return nil
}
