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

	g.Add("status", func(st *gostage.Stage) (string, error) {

		return st.Get("index"), nil
	})

	err := g.Run()

	if err != nil {

		fmt.Println(err)
	}

}
