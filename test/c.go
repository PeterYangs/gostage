package main

import (
	"github.com/joho/godotenv"
	"gostage"
)

func init() {

	err := godotenv.Load(".env")

	if err != nil {
		panic("配置文件加载失败")
	}
}

func main() {

	c := gostage.NewClient()

	c.Send("start")

}
