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
	Acquire() (interface{}, error) // 获取资源
	Release(interface{}) error     // 释放资源
	Close(interface{}) error       // 关闭资源
	Shutdown() error               // 关闭池
}

type PoolConfig struct {
	Min         int           // 池中最小对象数
	Max         int           // 池中最大对象数
	LiftTime    time.Duration // 对象生命周期
	FactoryFunc FactoryFunc   // 创建对象的工厂方法
	CloseFunc   CloseFunc     // 删除对象或关闭连接的方法
}

type GenericPool struct {
	sync.Mutex
	pool        chan interface{}
	maxCap      int           // 池中最大资源数
	minCap      int           // 池中最少资源数
	curNum      int           // 当前池中资源数
	closed      bool          // 池是否已关闭
	maxLifeTime time.Duration // 最大生命周期
	factoryFunc FactoryFunc   // 创建连接的方法
	closeFunc   CloseFunc     // 释放资源的方法
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
		// TODO maxLifeTime 处理
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
	// 新建连接
	obj, err := p.factoryFunc()
	if err != nil {
		p.Unlock()
		return nil, err
	}
	p.curNum++
	p.Unlock()
	return obj, nil
}

// 释放单个资源到连接池
func (p *GenericPool) Release(obj interface{}) error {
	if p.closed {
		return ErrPoolClosed
	}
	p.Lock()
	p.pool <- obj
	p.Unlock()
	return nil
}

// 关闭单个资源
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

// 关闭连接池，释放所有资源
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

// 查看当前池中资源的数量
func (p *GenericPool) Len() int {
	return len(p.pool)
}

func (p *GenericPool) IsClosed() bool {
	return p.closed
}
