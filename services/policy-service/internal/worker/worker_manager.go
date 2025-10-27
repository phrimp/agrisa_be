package worker

import (
	"context"
	"log"
	"sync"
)

type workerManagerCMDType int

const (
	StartPool workerManagerCMDType = iota
	StopPool
)

const (
	PoolTypeWorking   = "working"
	PoolTypeScheduler = "scheduler"
)

type WorkerManagerCMD struct {
	Type     workerManagerCMDType
	PoolName string
	PoolType string // Use PoolType constants
	Pool     Pool   // The pool to start (nil for StopPool)
}

type WorkerManager struct {
	workingPools   map[string]Pool
	poolCancels    map[string]context.CancelFunc // Used to stop individual pools
	wg             *sync.WaitGroup               // Can be understood as master pool, all pool created must wg.Add(1)
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
		workingPools:   make(map[string]Pool),
		poolCancels:    make(map[string]context.CancelFunc),
		cmdChan:        make(chan WorkerManagerCMD, 10), // Buffered command channel
	}
}

func (m *WorkerManager) Run() {
	log.Println("[Manager] Running...")
	defer log.Println("[Manager] Halted.")

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
			log.Println("[Manager] Shutdown signal received. Stopping all pools...")
			for name, cancel := range m.poolCancels {
				log.Printf("[Manager] Signaling pool '%s' to stop\n", name)
				cancel()
			}
			return
		}
	}
}

// this is add a new pool
func (m *WorkerManager) startPool(cmd WorkerManagerCMD) {
	if _, exists := m.poolCancels[cmd.PoolName]; exists {
		log.Printf("[Manager] Warning: Pool '%s' already exists.\n", cmd.PoolName)
		return
	}

	log.Printf("[Manager] Starting pool '%s' (Type: %s)\n", cmd.PoolName, cmd.PoolType)

	poolCtx, poolCancel := context.WithCancel(m.managerContext)

	m.poolCancels[cmd.PoolName] = poolCancel

	if cmd.PoolType == PoolTypeWorking {
		m.workingPools[cmd.PoolName] = cmd.Pool
	} else {
		m.workingPools[cmd.PoolName] = cmd.Pool
	}

	m.wg.Add(1)

	go cmd.Pool.Start(poolCtx, m.wg)
}

func (m *WorkerManager) stopPool(name string) {
	cancel, exists := m.poolCancels[name]
	if !exists {
		log.Printf("[Manager] Warning: Pool '%s' not found, cannot stop.\n", name)
		return
	}

	log.Printf("[Manager] Stopping pool '%s'\n", name)

	cancel()

	delete(m.poolCancels, name)
	delete(m.workingPools, name)

	// Note: The pool's Start() method is responsible for calling m.wg.Done() when it's fully stopped, so we dont need to do it here
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

func (m *WorkerManager) Shutdown() {
	log.Println("[Manager] Initiating shutdown...")
	m.managerCancel() // Signal Run() to stop

	m.wg.Wait() // wait for all pool to stop

	close(m.cmdChan)
	log.Println("[Manager] Shutdown complete.")
}
