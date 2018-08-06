package pool

import (
	"log"
	"math/rand"
	"runtime"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
)

func factory() (interface{}, error) {
	rand.Seed(time.Now().UnixNano())
	x := rand.Intn(100)
	return x, nil
}

func closer(o interface{}) error {
	o = -1
	log.Print("closer:", o)
	return nil
}

func clientFactory() (interface{}, error) {
	log.Println("[x] clientFactory")
	return &fasthttp.Client{}, nil
}

func clientCloser(i interface{}) error {
	return nil
}

func requestFactory() (interface{}, error) {
	log.Println("[x] requestFactory")
	return fasthttp.AcquireRequest(), nil
}

func requestCloser(i interface{}) error {
	fasthttp.ReleaseRequest(i.(*fasthttp.Request))
	return nil
}

func responseFactory() (interface{}, error) {
	log.Println("[x] responseFactory")
	return fasthttp.AcquireResponse(), nil
}

func responseCloser(i interface{}) error {
	fasthttp.ReleaseResponse(i.(*fasthttp.Response))
	return nil
}

var config = &PoolConfig{
	Min:         3,
	Max:         5,
	LiftTime:    5,
	FactoryFunc: clientFactory,
	CloseFunc:   clientCloser,
}

var requestConfig = &PoolConfig{
	Min:         3,
	Max:         5,
	LiftTime:    5,
	FactoryFunc: requestFactory,
	CloseFunc:   requestCloser,
}

var responseConfig = &PoolConfig{
	Min:         3,
	Max:         5,
	LiftTime:    5,
	FactoryFunc: responseFactory,
	CloseFunc:   responseCloser,
}

func TestNewGenericPool(t *testing.T) {
	pool, err := NewGenericPool(config)
	if err != nil {
		t.Log("[ERR]", err)
	}
	t.Log("[SUCC]", pool.Len())
}

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
	t.Logf("[SUCC] %T %+v", v1, v1.Object.(int))

	v2, _ := pool.Acquire()
	t.Logf("[SUCC] %T %+v", v2, v2.Object.(int))
	v3, _ := pool.Acquire()
	t.Logf("[SUCC] %T %+v", v3, v3.Object.(int))
	v4, err := pool.Acquire()
	if err != nil {
		t.Log("[ERR]", err)
	}
	t.Log("[SUCC]", pool.Len())
	t.Logf("[SUCC] %T %+v", v4, v4.Object.(int))
}

func TestGenericPool_Release(t *testing.T) {
	pool, err := NewGenericPool(config)
	if err != nil {
		t.Log("[ERR]", err)
	}
	t.Log("[SUCC]", pool.Len())

	v1, err := pool.Acquire()
	if err != nil {
		t.Log("[ERR]", err)
	}
	t.Log("[SUCC]", pool.Len())
	t.Logf("[SUCC] %T %+v", v1, v1.Object.(int))
	if err := pool.Release(v1); err != nil {
		t.Log("[ERR]", err)
	}
	t.Log("[SUCC]", pool.Len())
}

func TestGenericPool_Close(t *testing.T) {
	pool, err := NewGenericPool(config)
	if err != nil {
		t.Log("[ERR]", err)
	}
	t.Log("[SUCC]", pool.Len())

	v1, err := pool.Acquire()
	if err != nil {
		t.Log("[ERR]", err)
	}
	t.Log("[SUCC]", pool.Len())
	if err := pool.Close(v1); err != nil {
		t.Log("[ERR]", err)
	}
	t.Log("[SUCC]", pool.Len())
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

func TestFastHttpHostClient(t *testing.T) {
	c := fasthttp.HostClient{}
	statusCode, body, err := c.Get(nil, "http://www.google.com.hk")
	if err != nil {
		t.Log("[ERR]", err)
	}
	t.Log("[SUCC]", statusCode, string(body))
}

func TestFastHttpClient(t *testing.T) {
	request := fasthttp.AcquireRequest()
	response := fasthttp.AcquireResponse()
	request.SetRequestURI("http://www.google.com.hk")

	client := fasthttp.Client{}
	err := client.Do(request, response)
	if err != nil {
		t.Log("[ERR]", err)
	}
	t.Log("[SUCC]", string(response.Body()))
}

func TestFastHttpClient2(t *testing.T) {
	client := &fasthttp.Client{}
	statusCode, body, err := client.Get(nil, "http://www.google.com.hk")
	if err != nil {
		t.Log("[ERR]", err)
	}
	t.Log("[SUCC]", client, statusCode, string(body))

	statusCode2, body2, err2 := client.Get(nil, "http://www.google.com.hk")
	if err != nil {
		t.Log("[ERR]", err2)
	}
	t.Log("[SUCC]", client, statusCode2, string(body2))
}

func TestFastHttpClient3(t *testing.T) {
	var ms1, ms2, ms3 runtime.MemStats
	runtime.ReadMemStats(&ms1)
	pool, err := NewGenericPool(config)
	if err != nil {
		t.Log("[ERR]", err)
	}
	t.Log("[SUCC]", pool.Len())
	runtime.ReadMemStats(&ms2)

	t.Logf("1 : [Sys]: %d, [Alloc]: %d, [HeapSys]: %d, [HeapAlloc]: %d, [HeapInuse]: %d, [TotalAlloc]: %d", ms1.Sys, ms1.Alloc, ms1.HeapSys, ms1.HeapAlloc, ms1.HeapInuse, ms1.TotalAlloc)
	t.Logf("2 : [Sys]: %d, [Alloc]: %d, [HeapSys]: %d, [HeapAlloc]: %d, [HeapInuse]: %d, [TotalAlloc]: %d", ms2.Sys, ms2.Alloc, ms2.HeapSys, ms2.HeapAlloc, ms2.HeapInuse, ms2.TotalAlloc)
	for i := 0; i < 10; i++ {
		v, err := pool.Acquire()
		if err != nil {
			t.Log("[ERR]", err)
		}

		_, _, err = v.Object.(*fasthttp.Client).Get(nil, "http://www.google.com.hk")
		if err != nil {
			t.Log("[ERR]", err)
		}
		pool.Release(v)
		runtime.ReadMemStats(&ms3)
		t.Logf("3 : [Sys]: %d, [Alloc]: %d, [HeapSys]: %d, [HeapAlloc]: %d, [HeapInuse]: %d, [TotalAlloc]: %d", ms3.Sys, ms3.Alloc, ms3.HeapSys, ms3.HeapAlloc, ms3.HeapInuse, ms3.TotalAlloc)
	}

}

func TestFastHttpClient4(t *testing.T) {
	var ms1, ms2, ms3 runtime.MemStats
	runtime.ReadMemStats(&ms1)
	pool, err := NewGenericPool(config)
	if err != nil {
		t.Log("[ERR]", err)
	}
	t.Log("[SUCC]", pool.Len())
	reqPool, err := NewGenericPool(requestConfig)
	if err != nil {
		t.Log("[ERR]", err)
	}
	t.Log("[SUCC]", pool.Len())
	resPool, err := NewGenericPool(responseConfig)
	if err != nil {
		t.Log("[ERR]", err)
	}
	t.Log("[SUCC]", pool.Len())
	runtime.ReadMemStats(&ms2)

	t.Logf("1 : [Sys]: %d, [Alloc]: %d, [HeapSys]: %d, [HeapAlloc]: %d, [HeapInuse]: %d, [TotalAlloc]: %d", ms1.Sys, ms1.Alloc, ms1.HeapSys, ms1.HeapAlloc, ms1.HeapInuse, ms1.TotalAlloc)
	t.Logf("2 : [Sys]: %d, [Alloc]: %d, [HeapSys]: %d, [HeapAlloc]: %d, [HeapInuse]: %d, [TotalAlloc]: %d", ms2.Sys, ms2.Alloc, ms2.HeapSys, ms2.HeapAlloc, ms2.HeapInuse, ms2.TotalAlloc)
	for i := 0; i < 10; i++ {
		v, err := pool.Acquire()
		if err != nil {
			t.Log("[ERR]", err)
		}
		//t.Logf("[SUCC] %T %+v", v, v)

		req, _ := reqPool.Acquire()
		res, _ := resPool.Acquire()
		req.Object.(*fasthttp.Request).SetRequestURI("http://www.google.com.hk")
		err = v.Object.(*fasthttp.Client).Do(req.Object.(*fasthttp.Request), res.Object.(*fasthttp.Response))
		if err != nil {
			t.Log("[ERR]", err)
		}
		pool.Release(v)
		reqPool.Release(req)
		resPool.Release(res)
		runtime.ReadMemStats(&ms3)
		t.Logf("3 : [Sys]: %d, [Alloc]: %d, [HeapSys]: %d, [HeapAlloc]: %d, [HeapInuse]: %d, [TotalAlloc]: %d", ms3.Sys, ms3.Alloc, ms3.HeapSys, ms3.HeapAlloc, ms3.HeapInuse, ms3.TotalAlloc)
	}

}
