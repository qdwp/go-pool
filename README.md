# Generic Pool

## About

A package for Go language whitch you can create a generic pool using `interface{}` . So you can save any type object into that pool. 

## Install

```
	go get -u github.com/qdwp/go-pool
```

## Interface

```
type FactoryFunc func() (interface{}, error)
type CloseFunc func(interface{}) error

type Pool interface {
	Acquire() (interface{}, error) // acquire object from pool
	Release(interface{}) error     // release object from pool
	Close(interface{}) error       // close or delete object
	Shutdown() error               // shutdown current pool
}
```

## Use

```
// factory function
func factory() (interface{}, error) {
	rand.Seed(time.Now().UnixNano())
	x := rand.Intn(100)
	return x, nil
}

func closer(o interface{}) error {
	return nil
}

// init generic pool config
var config = &PoolConfig{
	Min:         3,
	Max:         5,
	LiftTime:    time.Second * 5,
	FactoryFunc: factory,
	CloseFunc:   closer,
}

```

## Test

```
func TestGenericPool_Acquire(t *testing.T) {
	pool, err := NewGenericPool(config)
	if err != nil {
		t.Log("[ERR]", err)
	}
	t.Log("[SUCC]", pool.Len())
	v1, err := pool.Acquire()
	if err != nil {
		t.Log("[ERR]", err)
	}
	t.Logf("[SUCC] %T %+v", v1, v1.(int))

	v2, _ := pool.Acquire()
	t.Logf("[SUCC] %T %+v", v2, v2.(int))
	v3, _ := pool.Acquire()
	t.Logf("[SUCC] %T %+v", v3, v3.(int))
	v4, err := pool.Acquire()
	if err != nil {
		t.Log("[ERR]", err)
	}
	t.Log("[SUCC]", pool.Len())
	t.Logf("[SUCC] %T %+v", v4, v4.(int))
}


func TestGenericPool_Shutdown(t *testing.T) {
	pool, err := NewGenericPool(config)
	if err != nil {
		t.Log("[ERR]", err)
	}
	t.Log("[SUCC]", pool.Len())

	if err := pool.Shutdown(); err != nil {
		t.Log("[ERR]", err)
	}
	t.Log("[SUCC]", pool.IsClosed())
	t.Log("[SUCC]", pool.Len())
}
```
