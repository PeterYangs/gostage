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

	g.StartFunc(func(st *gostage.Stage) error {

		index := 0

		for {

			select {

			case <-st.GetCxt().Done():

				return nil

			default:

				time.Sleep(1 * time.Second)

				fmt.Println(1111111111)

				index++

				st.Set("index", cast.ToString(index))

			}

		}

	})

	s := g.AddCommand("status", "当前进度.", func(st *gostage.Stage) (string, error) {

		//st.

		return st.Get("index"), nil
	})

	s.Flag("path", "文件地址.")

	s.Flag("name", "姓名.")

	err := g.Run()

	if err != nil {

		fmt.Println(err)
	}

}
