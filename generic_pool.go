package pool

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrInvalidConfig = errors.New("invalid pool config")
	ErrPoolClosed    = errors.New("pool is closed")
	ErrFactoryFunc   = errors.New("factory func err")
)

type FactoryFunc func() (interface{}, error)
type CloseFunc func(interface{}) error

type Pool interface {
	Acquire() (interface{}, error) // acquire object from pool
	Release(interface{}) error     // release object from pool
	Close(interface{}) error       // close or delete object
	Shutdown() error               // shutdown current pool
}

type PoolConfig struct {
	Min         int           // minimum objects of pool
	Max         int           // maximum objects of pool
	LiftTime    time.Duration // object's life tile
	FactoryFunc FactoryFunc   // function to new object
	CloseFunc   CloseFunc     // function to close or delete object
}

type GenericPool struct {
	sync.Mutex
	pool        chan interface{}
	maxCap      int           // max capacity of pool
	minCap      int           // min capacity of pool
	curNum      int           // current object number in pool
	closed      bool
	maxLifeTime time.Duration
	factoryFunc FactoryFunc
	closeFunc   CloseFunc
}

func NewGenericPool(config *PoolConfig) (*GenericPool, error) {
	if config.Max <= 0 || config.Min > config.Max {
		return nil, ErrInvalidConfig
	}
	p := &GenericPool{
		maxCap:      config.Max,
		minCap:      config.Min,
		maxLifeTime: config.LiftTime,
		factoryFunc: config.FactoryFunc,
		closeFunc:   config.CloseFunc,
		pool:        make(chan interface{}, config.Max),
	}

	for i := 0; i < p.minCap; i++ {
		obj, err := p.factoryFunc()
		if err != nil {
			continue
		}
		p.curNum++
		p.pool <- obj
	}
	if p.curNum == 0 {
		return p, ErrFactoryFunc
	}
	return p, nil
}

func (p *GenericPool) Acquire() (interface{}, error) {
	if p.closed {
		return nil, ErrPoolClosed
	}
	for {
		obj, err := p.getOrCreate()
		if err != nil {
			return nil, err
		}
		// TODO handle maxLifeTime
		return obj, nil
	}
}

func (p *GenericPool) getOrCreate() (interface{}, error) {
	select {
	case obj := <-p.pool:
		return obj, nil
	default:
	}
	p.Lock()
	if p.curNum >= p.maxCap {
		obj := <-p.pool
		p.Unlock()
		return obj, nil
	}
	// new object
	obj, err := p.factoryFunc()
	if err != nil {
		p.Unlock()
		return nil, err
	}
	p.curNum++
	p.Unlock()
	return obj, nil
}

// release object into pool
func (p *GenericPool) Release(obj interface{}) error {
	if p.closed {
		return ErrPoolClosed
	}
	p.Lock()
	p.pool <- obj
	p.Unlock()
	return nil
}

// close or delete object
func (p *GenericPool) Close(obj interface{}) error {
	p.Lock()
	if err := p.closeFunc(obj); err != nil {
		p.Unlock()
		return err
	}
	p.curNum--
	p.Unlock()
	return nil
}

// shutdown current pool, and remove all object from that pool
func (p *GenericPool) Shutdown() error {
	if p.closed {
		return ErrPoolClosed
	}
	p.Lock()
	close(p.pool)
	for obj := range p.pool {
		if err := p.closeFunc(obj); err != nil {
			p.Unlock()
			return err
		}
		p.curNum--
	}
	p.closed = true
	p.Unlock()
	return nil
}

// object numbers in current pool
func (p *GenericPool) Len() int {
	return len(p.pool)
}

func (p *GenericPool) IsClosed() bool {
	return p.closed
}
