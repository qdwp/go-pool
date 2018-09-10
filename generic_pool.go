package pool

import (
	"errors"
	"sync"
	"time"
	"fmt"
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

type PoolObject struct {
	CreateTime int64
	Object     interface{}
}

type GenericPool struct {
	sync.Mutex
	pool        chan PoolObject
	maxCap      int               // max capacity of pool
	minCap      int               // min capacity of pool
	curNum      int               // current object number in pool
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
		pool:        make(chan PoolObject, config.Max),
	}

	nowTime := time.Now().Unix()
	for i := 0; i < p.minCap; i++ {
		obj, err := p.factoryFunc()
		if err != nil {
			continue
		}
		p.curNum++
		poolObj := PoolObject{CreateTime: nowTime, Object: obj}
		p.pool <- poolObj
	}
	if p.curNum == 0 {
		return p, ErrFactoryFunc
	}
	return p, nil
}

func (p *GenericPool) isLiftTimeOut(obj PoolObject) bool {
	if int64(p.maxLifeTime) <= 0 {
		// if object is invalid
		return false
	}
	return obj.CreateTime+int64(p.maxLifeTime) <= time.Now().Unix()
}

func (p *GenericPool) Acquire() (poolObj PoolObject, err error) {
	if p.closed {
		return poolObj, ErrPoolClosed
	}
	for {
		poolObj, err = p.getOrCreate()
		if err != nil {
			fmt.Println("[POOL][ERROR] get or create object falied.")
			return poolObj, err
		}
		// handle maxLifeTime
		if p.isLiftTimeOut(poolObj) {
			continue
		}
		return poolObj, nil
	}
}

func (p *GenericPool) getOrCreate() (poolObj PoolObject, err error) {
	select {
	case poolObj = <-p.pool:
		return
	default:
	}
	p.Lock()
	if p.curNum >= p.maxCap {
		poolObj = <-p.pool
		p.Unlock()
		return
	}
	// new an object
	nowTime := time.Now().Unix()
	obj, err := p.factoryFunc()
	if err != nil {
		p.Unlock()
		return
	}
	p.curNum++
	poolObj.CreateTime = nowTime
	poolObj.Object = obj
	//poolObj = PoolObject{CreateTime: nowTime, Object: obj}
	p.Unlock()
	return
}

// release object into pool
func (p *GenericPool) Release(poolObj PoolObject) error {
	if p.closed {
		return ErrPoolClosed
	}
	if !p.isLiftTimeOut(poolObj) {
		p.Lock()
		p.pool <- poolObj
		p.Unlock()
	}
	return nil
}

// close or delete object
func (p *GenericPool) Close(poolObj PoolObject) error {
	p.Lock()
	if err := p.closeFunc(poolObj.Object); err != nil {
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
	for poolObj := range p.pool {
		if err := p.closeFunc(poolObj.Object); err != nil {
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
