package summer

import (
	"testing"
)

type test_server struct {
	Hello string
}

func TestContainer_Register(t *testing.T) {
	con := new(Container)
	con.Register(test_server{})
	if con.GetType("summer.test_server") == nil{
		t.Failed()
	}
}

func TestContainer_Empty(t *testing.T) {
	con := new(Container)
	if con.GetType("summer.test_server") != nil{
		t.Failed()
	}
	if con.Get("summer.test_server") != nil{
		t.Failed()
	}
}

func TestContainer_Get(t *testing.T) {
	con := new(Container)
	con.Register(test_server{})
	if con.Get("summer.test_server") == nil{
		t.Failed()
	}
}
