package gostage

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
)

type Server struct {
	sockFile string
	ctx      context.Context
	listen   net.Listener
	callback func(server *Server, param string, conn net.Conn)
}

func NewServer(cxt context.Context) *Server {

	return &Server{

		sockFile: os.Getenv("SOCK_FILE"),
		ctx:      cxt,
	}
}

func (s *Server) Start() error {

	listen, err := net.Listen("unix", s.sockFile)

	if err != nil {

		return err
	}

	s.listen = listen

	defer os.Remove(s.sockFile)

	go func() {

		select {

		case <-s.ctx.Done():

			s.listen.Close()

			return

		}

	}()

	go s.work()

	return nil
}

// StartWait 启动并等待
func (s *Server) StartWait() error {

	listen, err := net.Listen("unix", s.sockFile)

	if err != nil {

		return err
	}

	s.listen = listen

	defer os.Remove(s.sockFile)

	go func() {

		select {

		case <-s.ctx.Done():

			s.listen.Close()

			return

		}

	}()

	s.work()

	return nil

}

// Callback 回调赋值
func (s *Server) Callback(f func(server *Server, param string, conn net.Conn)) {

	s.callback = f
}

func (s *Server) work() {

	for {

		select {
		case <-s.ctx.Done():

			return

		default:

		}

		// 等待客户端建立连接
		conn, err := s.listen.Accept()

		if err != nil {
			fmt.Printf("accept failed, err:%v\n", err)
			continue
		}

		go s.read(conn)

	}
}

func (s *Server) read(conn net.Conn) {

	// 处理完关闭连接
	defer conn.Close()

	// 针对当前连接做发送和接受操作
	for {
		reader := bufio.NewReader(conn)
		var buf [1024]byte
		n, err := reader.Read(buf[:])
		if err != nil {
			//fmt.Printf("read from conn failed, err:%v\n", err)
			break
		}

		recv := string(buf[:n])

		if s.callback != nil {

			s.callback(s, recv, conn)
		}

	}
}
