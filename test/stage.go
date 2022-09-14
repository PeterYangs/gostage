package main

import (
	"context"
	"fmt"
	"github.com/PeterYangs/gostage"
	"time"
)

func main() {

	cxt, _ := context.WithCancel(context.Background())

	g := gostage.NewStage(cxt)

	//g.SetRunUser("nginx")

	s := g.StartFunc(func(request *gostage.Request) (string, error) {

		//index := 0

		fmt.Println("启动！")

		defer func() {

			fmt.Println("结束！")

			request.StopDaemonProcess()

		}()

		for {

			select {

			case <-request.GetCxt().Done():

				return "", nil

			default:

				//panic("异常测试")

				for i := 0; i < 100; i++ {

					time.Sleep(1 * time.Second)

					fmt.Println(i)

					//panic("异常测试")

					//return "", nil

				}

				panic("异常测试")

				return "nice", nil

				//index++

				//fmt.Println(request.GetFlag("file"))

				//request.Set("index", cast.ToString(index))

			}

		}

	})

	s.Flag("file", "文件路径.").Short('f')

	g.AddCommand("status", "当前进度.", func(request *gostage.Request) (string, error) {

		index, _ := request.Get("index")

		return index, nil
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
