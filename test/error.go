package main

import (
	"context"
	"fmt"
	"github.com/PeterYangs/gostage"
)

func main() {

	cxt, _ := context.WithCancel(context.Background())

	g := gostage.NewStage(cxt)

	g.StartFunc(func(request *gostage.Request) (string, error) {

		fmt.Println("启动！！")

		//time.Sleep(1000 * time.Second)
		//time.Sleep(3 * time.Second)
		//
		//panic("错误测试")

		for {

			select {

			case <-request.GetCxt().Done():

				return "", nil

			}
		}

		return "", nil
	})

	err := g.Run()

	if err != nil {

		fmt.Println(err)
	}

}
