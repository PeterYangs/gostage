package main

import (
	"context"
	"fmt"
	"github.com/PeterYangs/gostage"
	"github.com/spf13/cast"
	"time"
)

func main() {

	cxt, _ := context.WithCancel(context.Background())

	g := gostage.NewStage(cxt)

	g.LoadConfig(gostage.Config{
		RunUser: "nginx",
	})

	s := g.StartFunc(func(request *gostage.Request) (string, error) {

		index := 0

		for {

			select {

			case <-request.GetCxt().Done():

				return "", nil

			default:

				time.Sleep(1 * time.Second)

				index++

				fmt.Println(request.GetFlag("file"))

				request.Set("index", cast.ToString(index))

			}

		}

	})

	s.Flag("file", "文件路径.").Short('f')

	g.AddCommand("status", "当前进度.", func(request *gostage.Request) (string, error) {

		return request.Get("index"), nil
	})

	g.AddCommand("nice", "测试", func(request *gostage.Request) (string, error) {

		return "测试啊", nil
	}).NoConnect()

	//s.Flag("path", "文件地址.")
	//
	//s.Flag("name", "姓名.")
	//
	//s.Arg("filename", "文件名称.").Required()

	err := g.Run()

	if err != nil {

		fmt.Println(err)
	}

}
