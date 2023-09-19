package pack

import (
	"fmt"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestSocket(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(2)

	var testError error

	type objectA struct {
		String string
	}

	type objectB struct {
		String string
	}

	options := Options{
		WithObjects: NewObjects(
			objectA{},
			objectB{},
		),
	}

	// Server
	go func() {
		defer wg.Done()
		defer listener.Close()

		conn, err := listener.Accept()
		if err != nil {
			testError = err
			return
		}
		defer conn.Close()

		socketServer := NewSocket(conn, options)

		obj, err := socketServer.ReadTimeout(time.Second)
		if err != nil {
			testError = err
			return
		}

		if o, ok := obj.(*objectA); !ok {
			testError = fmt.Errorf("expected to receive *objectA, got %s", reflect.TypeOf(obj).String())
			return
		} else if o.String != "Hello, World!" {
			testError = fmt.Errorf("expected *objectA.String to be \"Hello, World!\", got %q", o.String)
			return
		}

		err = socketServer.WriteTimeout(objectB{
			String: "Hello, World!",
		}, time.Second)

		if err != nil {
			testError = err
			return
		}

	}()

	// Client
	go func() {
		defer wg.Done()
		defer listener.Close()

		conn, err := net.Dial("tcp", listener.Addr().String())
		if err != nil {
			testError = err
			return
		}
		defer conn.Close()

		socketClient := NewSocket(conn, options)

		err = socketClient.WriteTimeout(objectA{
			String: "Hello, World!",
		}, time.Second)

		if err != nil {
			testError = err
			return
		}

		obj, err := socketClient.ReadTimeout(time.Second)
		if err != nil {
			testError = err
			return
		}

		if o, ok := obj.(*objectB); !ok {
			testError = fmt.Errorf("expected to receive *objectA, got %s", reflect.TypeOf(obj).String())
			return
		} else if o.String != "Hello, World!" {
			testError = fmt.Errorf("expected *objectA.String to be \"Hello, World!\", got %q", o.String)
			return
		}
	}()

	wg.Wait()

	if testError != nil {
		t.Error(testError)
	}
}
