package gostage

import (
	"errors"
	"fmt"
	"io"
	"net"
)

type Client struct {
	sockFile string
}

func NewClient(st *Stage) *Client {

	return &Client{
		sockFile: st.getSockName(),
	}
}

func (c *Client) Send(param string) (string, error) {

	conn, err := net.Dial("unix", c.sockFile)

	if err != nil {

		return "", errors.New(fmt.Sprintf("客户端连接失败, err:%v", err))
	}

	defer conn.Close()

	_, err = conn.Write([]byte(param + "\n"))

	if err != nil {

		return "", errors.New(fmt.Sprintf("客户端数据发送失败, err:%v", err))
	}

	res := ""

	for {

		var buf = make([]byte, 1024)

		n, e := conn.Read(buf)

		if e != nil {

			if e != io.EOF {

				return "", errors.New(fmt.Sprintf("读取服务端失败, err:%v", e))

			}

			break

		}

		res += string(buf[:n])

	}

	return res, nil

}
