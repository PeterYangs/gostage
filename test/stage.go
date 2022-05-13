package main

import (
	"context"
	"fmt"
	"gostage"
	"time"
)

func main() {

	cxt, cancel := context.WithCancel(context.Background())

	g := gostage.NewStage(cxt, cancel)

	g.StartFunc(func(st *gostage.Stage) error {

		for {

			select {

			case <-st.GetCxt().Done():

				return nil

			default:

				time.Sleep(1 * time.Second)

				fmt.Println(1111111111)

			}

		}

	})

	err := g.Run()

	if err != nil {

		fmt.Println(err)
	}

}
