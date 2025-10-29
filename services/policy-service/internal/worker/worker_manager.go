package worker

import (
	"context"
	"log/slog"
	"sync"
)

type workerManagerCMDType int

const (
	StartPool workerManagerCMDType = iota
	StopPool
)

// PoolType constants
const (
	PoolTypeWorking = "working"
)

// WorkerManagerCMD is the command struct sent to the manager's loop.
type WorkerManagerCMD struct {
	Type     workerManagerCMDType
	PoolName string
	PoolType string
	Pool     Pool
}

// WorkerManager controls the entire lifecycle of all pools.
type WorkerManager struct {
	pools       map[string]Pool
	poolCancels map[string]context.CancelFunc
	wg          *sync.WaitGroup
	mu          sync.RWMutex

	managerContext context.Context
	managerCancel  context.CancelFunc
	cmdChan        chan WorkerManagerCMD
}

func NewWorkerManager() *WorkerManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerManager{
		wg:             new(sync.WaitGroup),
		managerContext: ctx,
		managerCancel:  cancel,
		pools:          make(map[string]Pool),
		poolCancels:    make(map[string]context.CancelFunc),
		cmdChan:        make(chan WorkerManagerCMD, 10),
	}
}

func (m *WorkerManager) ManagerContext() context.Context {
	return m.managerContext
}

func (m *WorkerManager) Run() {
	slog.Info("Worker manager starting")
	defer slog.Info("Worker manager halted")

	for {
		select {
		case cmd := <-m.cmdChan:
			switch cmd.Type {
			case StartPool:
				m.startPool(cmd)
			case StopPool:
				m.stopPool(cmd.PoolName)
			}
		case <-m.managerContext.Done():
			slog.Info("Worker manager shutdown signal received, stopping all pools")
			for name, cancel := range m.poolCancels {
				slog.Info("Signaling pool to stop", "pool_name", name)
				cancel()
			}
			return
		}
	}
}

func (m *WorkerManager) startPool(cmd WorkerManagerCMD) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.poolCancels[cmd.PoolName]; exists {
		slog.Warn("Pool already exists, skipping start", "pool_name", cmd.PoolName, "pool_type", cmd.PoolType)
		return
	}
	slog.Info("Starting pool", "pool_name", cmd.PoolName, "pool_type", cmd.PoolType)
	poolCtx, poolCancel := context.WithCancel(m.managerContext)
	m.poolCancels[cmd.PoolName] = poolCancel
	m.pools[cmd.PoolName] = cmd.Pool
	m.wg.Add(1)
	go cmd.Pool.Start(poolCtx, m.wg)
}

func (m *WorkerManager) stopPool(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cancel, exists := m.poolCancels[name]
	if !exists {
		slog.Warn("Pool not found, cannot stop", "pool_name", name)
		return
	}
	slog.Info("Stopping pool", "pool_name", name)
	cancel()
	delete(m.poolCancels, name)
	delete(m.pools, name)
}

func (m *WorkerManager) StartPool(name string, pool Pool, poolType string) {
	cmd := WorkerManagerCMD{
		Type:     StartPool,
		PoolName: name,
		PoolType: poolType,
		Pool:     pool,
	}
	m.cmdChan <- cmd
}

func (m *WorkerManager) StopPool(name string) {
	cmd := WorkerManagerCMD{
		Type:     StopPool,
		PoolName: name,
	}
	m.cmdChan <- cmd
}

func (m *WorkerManager) GetPool(name string) (Pool, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pool, exists := m.pools[name]
	return pool, exists
}

func (m *WorkerManager) Shutdown() {
	slog.Info("Worker manager initiating shutdown")
	m.managerCancel()
	m.wg.Wait()
	close(m.cmdChan)
	slog.Info("Worker manager shutdown complete")
}
