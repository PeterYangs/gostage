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

				//fmt.Println(request.GetFlag("file"))

				index++

				fmt.Println(index)

				request.Set("index", cast.ToString(index))

				//cmd := exec.CommandContext(cxt, "/bin/bash", "-c", "")

				//cmd.SysProcAttr.ProcessAttributes.SecurityDescriptor

			}

		}

	})

	s.Flag("file", "文件路径.").Short('f')

	g.AddCommand("status", "当前进度.", func(request *gostage.Request) (string, error) {

		return request.Get("index"), nil
	})

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
