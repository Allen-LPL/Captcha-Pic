package main

import (
	"captcha-pic/controllers"
	"fmt"
	"github.com/hprose/hprose-golang/rpc"
)

func hello(name string) string {
	return "Hello " + name + "!"
}

func main() {
	service := rpc.NewTCPServer("tcp4://0.0.0.0:1918/")
	service.AddFunction("hello", hello)
	//注册struct，命名空间是Sample
	service.AddInstanceMethods(&controllers.CaptchaController{}, rpc.Options{NameSpace: "Captcha"})
	//service.AddAllMethods(&controllers.CaptchaController{})
	//service.AddInstanceMethods(&controllers.CaptchaController{}, rpc.Options{NameSpace: "Captcha"})
	err := service.Start()
	if err != nil {
		fmt.Printf("start server fail, err:%v\n", err)
		return
	}
}
