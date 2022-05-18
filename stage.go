package gostage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PeterYangs/gcmd2"
	"github.com/PeterYangs/gostage/lib/kill"
	"github.com/PeterYangs/tools"
	"github.com/PeterYangs/tools/file/read"
	"github.com/joho/godotenv"
	"github.com/spf13/cast"
	"gopkg.in/alecthomas/kingpin.v2"
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
	list      map[string]*item
	startFunc func(st *Stage) error
	wait      sync.WaitGroup
	lock      sync.Mutex
	data      map[string]string
	appDesc   string
}

type item struct {
	fun   func(request *Request) (string, error)
	flags []*Flag
	args  []*Arg
	name  string
	st    *Stage
	help  string
}

type data struct {
	Name  string            `json:"name"`
	Flags map[string]string `json:"flags"`
	Args  map[string]string `json:"args"`
}

type Request struct {
	name  string
	flags map[string]string
	args  map[string]string
	st    *Stage
	conn  net.Conn
	lock  sync.Mutex
}

func NewRequest(st *Stage, name string, flags map[string]string, args map[string]string, conn net.Conn) *Request {

	return &Request{name: name, flags: flags, args: args, st: st, conn: conn, lock: sync.Mutex{}}
}

func (request *Request) Get(key string) string {

	return request.st.Get(key)
}

func (request *Request) Set(key string, value string) {

	request.st.Set(key, value)
}

func (request *Request) GetFlag(key string) string {

	request.lock.Lock()

	defer request.lock.Unlock()

	return request.flags[key]

}

func (request *Request) GetFlags() map[string]string {

	return request.flags
}

func (request *Request) GetArg(key string) string {

	request.lock.Lock()

	defer request.lock.Unlock()

	return request.args[key]

}

func (request *Request) GetArgs() map[string]string {

	return request.args
}

func NewItem(fun func(request *Request) (string, error), st *Stage, help string) *item {

	return &item{fun: fun, st: st, help: help, flags: []*Flag{}, args: []*Arg{}}
}

func (i *item) Flag(name string, help string) *Flag {

	f := NewFlag(name, help)

	i.flags = append(i.flags, f)

	return f

}

func (i *item) Arg(name string, help string) *Arg {

	a := NewArg(name, help)

	i.args = append(i.args, a)

	return a
}

type Flag struct {
	name     string
	value    string
	help     string
	required bool
}

func NewFlag(name string, help string) *Flag {

	return &Flag{name: name, help: help}
}

func (flag *Flag) Required() *Flag {

	flag.required = true

	return flag

}

type Arg struct {
	name     string
	value    string
	help     string
	required bool
}

func NewArg(name string, help string) *Arg {

	return &Arg{name: name, help: help}
}

func (arg *Arg) Required() *Arg {

	arg.required = true

	return arg

}

func NewStage(cxt context.Context) *Stage {

	ct, cancel := context.WithCancel(cxt)

	return &Stage{ctx: ct, cancel: cancel, wait: sync.WaitGroup{}, lock: sync.Mutex{}, data: make(map[string]string, 0), list: make(map[string]*item)}
}

func (st *Stage) AddCommand(param string, help string, f func(request *Request) (string, error)) *item {

	i := NewItem(f, st, help)

	st.list[param] = i

	return i

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

		serv := NewServer(st)

		serv.Callback(func(server *Server, param string, conn net.Conn, flags map[string]string, args map[string]string) {

			defer conn.Close()

			switch param {

			case "stop":

				st.cancel()

			default:

				for s, f2 := range st.list {

					if param == s {

						fmt.Println("参数为:", s)

						request := NewRequest(st, param, flags, args, conn)

						msg, cErr := f2.fun(request)

						if cErr != nil {

							conn.Write([]byte(cErr.Error()))

							return
						}

						conn.Write([]byte(msg))

					}

				}

			}

		})

		//启动tcp服务器
		err := serv.Start()

		if err != nil {

			return err
		}

		defer st.cancel()

		err = f(st)

		return err

	}
}

func (st *Stage) Set(key string, value string) {

	st.lock.Lock()

	defer st.lock.Unlock()

	st.data[key] = value

}

func (st *Stage) Get(key string) string {

	return st.data[key]
}

func (st *Stage) setAppDesc(desc string) *Stage {

	st.appDesc = desc

	return st
}

func (st *Stage) Run() error {

	envErr := godotenv.Load(".env")

	if envErr != nil {

		return errors.New("配置文件加载失败:" + envErr.Error())
	}

	args := os.Args

	app := kingpin.New(args[0], st.appDesc)

	//启动
	start := app.Command("start", "启动服务.")

	start.Flag("daemon", "后台运行.").Short('d').Bool()

	//停止
	app.Command("stop", "停止运行.")

	//守护进程
	app.Command("daemon", "守护进程").Hidden()

	//绑定自定义命令
	for s, i := range st.list {

		cd := app.Command(s, i.help)

		for _, arg := range i.args {

			ag := cd.Arg(arg.name, arg.help)

			//必填
			if arg.required {

				ag.Required()
			}

			ag.String()

		}

		for _, flag := range i.flags {

			fg := cd.Flag(flag.name, flag.help)

			//必填
			if flag.required {

				fg.Required()
			}

			fg.String()
		}

	}

	if len(args) == 1 {

		if st.startFunc != nil {

			err := st.startFunc(st)

			st.wait.Wait()

			fmt.Println("finish!!!")

			return err

		}

		return errors.New("启动回调函数未设置")
	}

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {

	case "start":

		if st.startFunc != nil {

			if app.GetCommand("start").GetFlag("daemon").Model().String() == "true" {

				sysType := runtime.GOOS

				var cmd *gcmd2.Gcmd2

				runUser := os.Getenv("RUN_USER")

				if sysType == `linux` && runUser != "nobody" && runUser != "" {

					//以其他用户运行服务，源命令(sudo -u nginx ./main start)
					cmd = gcmd2.NewCommand("sudo -u "+runUser+" "+args[0]+" daemon"+" ", context.TODO())

				} else {

					cmd = gcmd2.NewCommand(args[0]+" daemon", context.TODO())

				}

				cErr := cmd.StartNoWait()

				fmt.Println(tools.Join(" ", args) + " ")

				return cErr

			}

			err := st.startFunc(st)

			if err != nil {

				return errors.New("启动失败:" + err.Error())
			}

			st.wait.Wait()

			fmt.Println("finish!!!")

			return nil

		}

		return errors.New("启动回调函数未设置")

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

	default:

		findArgs := false

		for s, i := range st.list {

			if args[1] == s {

				findArgs = true

				c := NewClient()

				//fmt.Println(app.GetCommand(s).Model().Flags)

				d := data{
					Name:  s,
					Flags: map[string]string{},
					Args:  map[string]string{},
				}

				//参数绑定
				for i2, flag := range i.flags {

					d.Flags[flag.name] = app.GetCommand(s).Model().Flags[i2].String()

				}

				for i2, arg := range i.args {

					d.Args[arg.name] = app.GetCommand(s).Model().Args[i2].String()

				}

				str, err := json.Marshal(d)

				if err != nil {

					fmt.Println(err)

					return errors.New("打包json失败:" + err.Error())

				}

				msg, cErr := c.Send(string(str))

				if cErr != nil {

					return cErr
				}

				fmt.Println(msg)

			}
		}

		if !findArgs {

			fmt.Println("未找到该命令")
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
