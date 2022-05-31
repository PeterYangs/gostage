package main

import (
	"context"
	"fmt"
	"github.com/PeterYangs/gostage"
	"os"
	"time"
)

func main() {

	g := gostage.NewStage(context.Background())

	//g.LoadConfig(gostage.Config{
	//	RunUser: "www",  //守护进程模式的执行用户（仅支持linux平台）
	//	RunPath: "run",  //运行文件(pid、sock等文件)的存放路径
	//	LogPath: "logs", //日志文件路径
	//})

	g.SetRunUser("nginx")

	//绑定主服务逻辑
	g.StartFunc(func(request *gostage.Request) (string, error) {

		//打开文件
		file, err := os.OpenFile("word.txt", os.O_CREATE|os.O_RDWR, 0644)

		if err != nil {

			return "", err
		}

		//计数
		index := 0

		defer file.Close()

		for {

			select {

			case <-request.GetCxt().Done():

				return "", nil

			default:

				//打印计数到终端
				fmt.Println(index)

				//每秒写入一行文本
				time.Sleep(1 * time.Second)

				file.Write([]byte("word\n"))

				index++

			}

		}

	})

	err := g.Run()

	if err != nil {

		fmt.Println(err)
	}

}
