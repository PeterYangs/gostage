package gostage

import (
	"context"
	"errors"
	"fmt"
	"github.com/PeterYangs/gcmd2"
	"github.com/PeterYangs/tools"
	"github.com/PeterYangs/tools/file/read"
	"github.com/joho/godotenv"
	"github.com/spf13/cast"
	"gostage/lib/kill"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"sync"
	"syscall"
	"time"
)

type Stage struct {
	ctx       context.Context
	cancel    context.CancelFunc
	server    *Server
	list      map[string]func(st *Stage)
	startFunc func(st *Stage) error
	wait      sync.WaitGroup
}

func NewStage(cxt context.Context, cancel context.CancelFunc) *Stage {

	return &Stage{ctx: cxt, cancel: cancel, wait: sync.WaitGroup{}}
}

func (st *Stage) Add(param string, f func(st *Stage)) {

	st.list[param] = f

}

func (st *Stage) StartFunc(f func(st *Stage) error) {

	st.startFunc = func(st *Stage) error {

		sErr := st.savePid()

		if sErr != nil {

			return errors.New("记录pid失败:" + sErr.Error())
		}

		defer os.Remove(os.Getenv("PID_FILE"))

		sigs := make(chan os.Signal, 1)

		go func(sta *Stage) {

			select {
			case <-sigs:

				fmt.Println("检查到退出信号")

				sta.cancel()

				break

			case <-sta.GetCxt().Done():

				break

			}

		}(st)

		//退出信号
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		serv := NewServer(st.ctx)

		serv.Callback(func(server *Server, param string, conn net.Conn) {

			defer conn.Close()

			switch param {

			case "stop":

				st.cancel()

			}

		})

		err := serv.Start()

		if err != nil {

			return err
		}

		defer st.cancel()

		err = f(st)

		return err

	}
}

func (st *Stage) Run() error {

	envErr := godotenv.Load(".env")

	if envErr != nil {

		return errors.New("配置文件加载失败:" + envErr.Error())
	}

	args := os.Args

	if len(args) == 1 {

		if st.startFunc != nil {

			err := st.startFunc(st)

			fmt.Println("finish!!!")

			return err

		}

		return errors.New("启动回调函数未设置")
	}

	switch args[1] {

	case "start":

		if st.startFunc != nil {

			daemon := false
			for k, v := range args {
				if v == "-d" {
					daemon = true
					args[k] = ""
				}
			}

			if daemon {

				args[1] = "daemon"

				cmd := gcmd2.NewCommand(tools.Join(" ", args)+" ", context.TODO())

				cErr := cmd.StartNoWait()

				return cErr

			}

			err := st.startFunc(st)

			if err != nil {

				return errors.New("启动失败:" + err.Error())
			}

			fmt.Println("finish!!!")

			return nil

		}

		return errors.New("启动回调函数未设置")

	case "daemon":

		//记录守护进程pid
		f, err := os.OpenFile("daemon.pid", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0664)

		if err != nil {

			return err
		}

		sigs := make(chan os.Signal, 1)

		//退出信号
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		defer func() {

			_ = os.Remove("daemon.pid")

		}()

		_, err = f.Write([]byte(cast.ToString(os.Getpid())))

		_ = f.Close()

		args[1] = "start"

		for {

			select {
			case <-sigs:

				return nil

			default:

				cmd := gcmd2.NewCommand(tools.Join(" ", args)+" ", context.TODO())

				st.dealOut(cmd)

				cErr := cmd.StartNotOut()

				if cErr != nil {

					return cErr
				}

			}

		}

		//return cErr

	case "stop":

		fmt.Println("stopping")

		err := st.stop()

		if err != nil {

			return err
		}

		//检测pid文件是否存在来判断程序是否还在运行
		for {

			time.Sleep(300 * time.Millisecond)

			ok, _ := PathExists(os.Getenv("PID_FILE"))

			if ok == false {

				fmt.Println("stopped")

				return nil
			}

		}

	}

	return nil
}

func (st *Stage) GetCxt() context.Context {

	return st.ctx
}

//------------------------------------------------------------------------------------

func (st *Stage) stop() error {

	isD, _ := PathExists("daemon.pid")

	if isD {

		//守护进程关闭

		sysType := runtime.GOOS

		dPid, err := read.Open("daemon.pid").Read()

		if err != nil {

			return err

		}

		if sysType == `windows` {

			//fmt.Println("")

			if st.createWindowsKill() {

				g := gcmd2.NewCommand("kill.exe -SIGINT "+string(dPid), context.Background())

				err := g.Start()

				if err != nil {

					return err
				}

				return st.sendStop()

			} else {

				return errors.New("windows生成kill.exe失败")
			}

		}

		if sysType == `linux` {

			g := gcmd2.NewCommand("kill  "+string(dPid), context.Background())

			err := g.Start()

			if err != nil {

				return err
			}

			return st.sendStop()

		}

	} else {

		//非守护进程关闭

		return st.sendStop()

	}

	return nil
}

func (st *Stage) sendStop() error {

	c := NewClient()

	_, err := c.Send("stop")

	if err != nil {

		return err
	}

	return nil
}

//记录主程序pid
func (st *Stage) savePid() error {

	f, err := os.OpenFile(os.Getenv("PID_FILE"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0664)

	if err != nil {

		return err
	}

	//记录pid
	_, err = f.Write([]byte(cast.ToString(os.Getpid())))

	_ = f.Close()

	return err

}

func (st *Stage) dealOut(g *gcmd2.Gcmd2) {

	outIo, err := g.GetOutPipe()

	if err != nil {

		return
	}

	errIo, err := g.GetErrPipe()

	if err != nil {

		return
	}

	go st.out(outIo)
	go st.err(errIo)

}

func (st *Stage) out(stt io.ReadCloser) {

	defer stt.Close()

	buf := make([]byte, 1024)

	f, fErr := os.OpenFile("outLog.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)

	if fErr != nil {

		log.Println(fErr, string(debug.Stack()))

		return
	}

	defer f.Close()

	for {

		n, readErr := stt.Read(buf)

		if readErr != nil {

			if readErr == io.EOF {

				return
			}

			return
		}

		fmt.Print(string(buf[:n]))

		f.Write(buf[:n])

	}

}

func (st *Stage) err(stt io.ReadCloser) {

	defer stt.Close()

	buf := make([]byte, 1024)

	f, fErr := os.OpenFile("outErr.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)

	if fErr != nil {

		log.Println(fErr, string(debug.Stack()))

		return
	}

	defer f.Close()

	for {

		n, readErr := stt.Read(buf)

		if readErr != nil {

			if readErr == io.EOF {

				return
			}

			return
		}

		fmt.Print(string(buf[:n]))

		f.Write(buf[:n])

	}
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (st *Stage) createWindowsKill() bool {

	b, err := PathExists("kill.exe")

	if err != nil {

		return false
	}

	if b {

		return true
	}

	f, err := os.OpenFile("kill.exe", os.O_CREATE|os.O_RDWR, 0755)

	if err != nil {

		return false
	}

	defer f.Close()

	f.Write(kill.Kill)

	return true

}
