package worker

import (
	"context"
	"fmt"
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
	fmt.Println("[Manager] Running...")
	defer fmt.Println("[Manager] Halted.")

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
			fmt.Println("[Manager] Shutdown signal received. Stopping all pools...")
			for name, cancel := range m.poolCancels {
				fmt.Printf("[Manager] Signaling pool '%s' to stop\n", name)
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
		fmt.Printf("[Manager] Warning: Pool '%s' already exists.\n", cmd.PoolName)
		return
	}
	fmt.Printf("[Manager] Starting pool '%s'\n", cmd.PoolName)
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
		fmt.Printf("[Manager] Warning: Pool '%s' not found.\n", name)
		return
	}
	fmt.Printf("[Manager] Stopping pool '%s'\n", name)
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
	fmt.Println("[Manager] Initiating shutdown...")
	m.managerCancel()
	m.wg.Wait()
	close(m.cmdChan)
	fmt.Println("[Manager] Shutdown complete.")
}
