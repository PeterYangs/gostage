package main

import (
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"gostage"
	"net"
)

func init() {

	err := godotenv.Load(".env")

	if err != nil {
		panic("配置文件加载失败")
	}
}

func main() {

	s := gostage.NewServer(context.Background())

	s.Callback(func(server *gostage.Server, param string, conn net.Conn) {

		defer conn.Close()

		fmt.Println(param)

	})

	err := s.StartWait()

	if err != nil {

		fmt.Println(err)

		return
	}

}
