package main

import (
	"context"
	"fmt"
	"github.com/PeterYangs/gostage"
	"os"
	"strconv"
	"time"
)

func main() {

	g := gostage.NewStage(context.Background())

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

				//存储计数到常驻变量
				request.Set("length", strconv.Itoa(index))

				//每秒写入一行文本
				time.Sleep(1 * time.Second)

				file.Write([]byte("word\n"))

				index++

			}

		}

	})

	g.AddCommand("length", "获取当前计数.", func(request *gostage.Request) (string, error) {

		return "当前计数为：" + request.Get("length"), nil
	})

	err := g.Run()

	if err != nil {

		fmt.Println(err)
	}

}
