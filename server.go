package gostage

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
)

type Server struct {
	sockFile string
	ctx      context.Context
	listen   net.Listener
	callback func(server *Server, param string, conn net.Conn, flags map[string]string, args map[string]string)
	st       *Stage
}

func NewServer(st *Stage) *Server {

	return &Server{

		sockFile: os.Getenv("SOCK_FILE"),
		ctx:      st.GetCxt(),
		st:       st,
	}
}

func (s *Server) Start() error {

	//fmt.Println(os.Getenv("SOCK_FILE"))

	listen, err := net.Listen("unix", s.sockFile)

	if err != nil {

		return err
	}

	s.st.wait.Add(1)

	s.listen = listen

	go func() {

		select {

		case <-s.ctx.Done():

			s.listen.Close()

			os.Remove(s.sockFile)

			s.st.wait.Done()

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
func (s *Server) Callback(f func(server *Server, param string, conn net.Conn, flags map[string]string, args map[string]string)) {

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

			select {
			case <-s.ctx.Done():

				continue

			default:

			}

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

		line, err := reader.ReadSlice('\n')
		if err != nil {
			break
		}

		var d data

		jErr := json.Unmarshal(line, &d)

		if jErr != nil {

			conn.Write([]byte("解析数据失败:" + jErr.Error()))

			return
		}

		if s.callback != nil {

			s.callback(s, d.Name, conn, d.Flags, d.Args)
		}

	}
}
