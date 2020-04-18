package main

import (
	"github.com/hprose/hprose-golang/rpc"
	"testing"
)
//stub：申明服务里拥有的方法
type clientStub struct {
	Hello       func(string) string
}
//获取一个客户端
func GetClient() *rpc.TCPClient {
	return rpc.NewTCPClient("tcp4://172.16.97.219:1314")
}

//测试普通方法
func TestHello(t *testing.T) {
	client := GetClient()

	defer client.Close()
	var stub clientStub
	client.UseService(&stub)

	rep := stub.Hello("func")
	if rep == "" {
		t.Error(rep)
	} else {
		t.Log(rep)
	}
}

