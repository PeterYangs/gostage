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
	list      []*item
	startFunc func(request *Request) (string, error)
	wait      sync.WaitGroup
	lock      sync.Mutex
	data      map[string]string
	appDesc   string
	config    *Config
}

type Config struct {
	RunPath string //pid存放路径
	RunUser string //运行用户
	LogPath string //日志存放路径
}

type data struct {
	Name  string            `json:"name"`
	Flags map[string]string `json:"flags"`
	Args  map[string]string `json:"args"`
}

func NewStage(cxt context.Context) *Stage {

	config := &Config{
		RunPath: "run",
		RunUser: "nobody",
		LogPath: "logs",
	}

	ct, cancel := context.WithCancel(cxt)

	return &Stage{ctx: ct, cancel: cancel, wait: sync.WaitGroup{}, lock: sync.Mutex{}, data: make(map[string]string, 0), list: make([]*item, 0), config: config}
}

func (st *Stage) SetRunUser(user string) *Stage {

	st.config.RunUser = user

	return st

}

func (st *Stage) SetRunPath(runPath string) *Stage {

	st.config.RunPath = runPath

	return st
}

func (st *Stage) SetLogPath(logPath string) *Stage {

	st.config.LogPath = logPath

	return st
}

func (st *Stage) AddCommand(param string, help string, f func(request *Request) (string, error)) *item {

	i := NewItem(param, f, st, help)

	st.list = append(st.list, i)

	return i

}

func (st *Stage) StartFunc(f func(request *Request) (string, error)) *item {

	st.startFunc = func(request *Request) (string, error) {

		sErr := st.savePid()

		if sErr != nil {

			return "", errors.New("记录pid失败:" + sErr.Error())
		}

		defer os.Remove(st.getRunPidName())

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

			case "start":

			case "stop":

				st.cancel()

			default:

				for _, f2 := range st.list {

					if param == f2.name {

						rt := NewRequest(st, param, flags, args)

						msg, cErr := f2.fun(rt)

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

			return "", err
		}

		defer st.cancel()

		return f(request)

	}

	i := NewItem("start", st.startFunc, st, "启动服务.")

	i.Flag("daemon", "后台运行.").Short('d').Bool()

	st.list = append(st.list, i)

	return i

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

//权限写入检查
func (st *Stage) permissionCheck() {

	fErr := st.pathDeal(st.config.RunPath)

	if fErr != nil {

		panic("运行文件夹写入权限检查失败:" + fErr.Error())

	}

	fErr = st.pathDeal(st.config.LogPath)

	if fErr != nil {

		panic("日志文件夹写入权限检查失败:" + fErr.Error())

	}

}

func (st *Stage) getRequest(app *kingpin.Application, param string) *Request {

	flags := make(map[string]string)
	args := make(map[string]string)

	startItem, _ := st.getItemByName(param)

	//参数绑定
	for i2, flag := range startItem.flags {

		flags[flag.name] = app.GetCommand(param).Model().Flags[i2].String()

	}

	for i2, arg := range startItem.args {

		args[arg.name] = app.GetCommand(param).Model().Args[i2].String()

	}

	return NewRequest(st, param, flags, args)

}

func (st *Stage) Run() error {

	//日志目录和运行目录权限检查
	st.permissionCheck()

	args := os.Args

	app := kingpin.New(args[0], st.appDesc)

	//内置命令
	st.requiredCommand()

	//绑定自定义命令
	for _, i := range st.list {

		cd := app.Command(i.name, i.help)

		if i.hide {

			cd.Hidden()
		}

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

			if flag.short != 0 {

				fg.Short(flag.short)
			}

			if flag.isBool {

				fg.Bool()

			} else {

				fg.String()
			}

		}

	}

	//停止
	app.Command("stop", "停止运行.")

	if len(args) == 1 {

		args = append(args, "start")

	}

	switch kingpin.MustParse(app.Parse(args[1:])) {

	case "start":

		if st.startFunc != nil {

			sigs := make(chan os.Signal, 1)

			//退出信号
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

			runUser := st.config.RunUser

			if app.GetCommand("start").GetFlag("daemon").Model().String() == "true" {

				cmd := gcmd2.NewCommand(args[0]+" daemon "+tools.Join(" ", args[2:]), context.TODO())

				if runUser != "nobody" && runUser != "" {

					//以其他用户运行服务
					cmd.SetUser(runUser)

				}

				cErr := cmd.StartNoWaitOutErr()

				time.Sleep(1 * time.Second)

				return cErr

			}

			args[1] = "run"

			cmd := gcmd2.NewCommand(tools.Join(" ", args)+" ", context.TODO())

			if runUser != "nobody" && runUser != "" {

				//以其他用户运行服务
				cmd.SetUser(runUser)

			}

			//给子进程发送退出信号
			go func(s chan os.Signal, c *gcmd2.Gcmd2) {

				select {

				case <-s:

					_ = c.GetCmd().Process.Signal(syscall.SIGINT)

					break

				}

			}(sigs, cmd)

			st.dealOut(cmd)

			cErr := cmd.StartNotOut()

			if cErr != nil {

				return cErr
			}

			return nil

		}

		return errors.New("启动回调函数未设置")

	case "run":

		res, err := st.startFunc(st.getRequest(app, "run"))

		if err != nil {

			return errors.New("启动失败:" + err.Error())
		}

		st.wait.Wait()

		fmt.Println(res)

		fmt.Println("finish!!!")

	case "stop":

		fmt.Println("stopping")

		err := st.stop()

		if err != nil {

			return err
		}

		//检测pid文件是否存在来判断程序是否还在运行
		for {

			time.Sleep(300 * time.Millisecond)

			ok, _ := PathExists(st.getRunPidName())

			if ok == false {

				fmt.Println("stopped")

				return nil
			}

		}

	case "daemon":

		//记录守护进程pid
		f, err := os.OpenFile(st.getDaemonName(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0664)

		if err != nil {

			return err
		}

		sigs := make(chan os.Signal, 1)

		//退出信号
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		defer func() {

			_ = os.Remove(st.getDaemonName())

		}()

		_, err = f.Write([]byte(cast.ToString(os.Getpid())))

		_ = f.Close()

		args[1] = "run"

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

		for _, i := range st.list {

			if args[1] == i.name {

				findArgs = true

				if i.noConnect {

					res, err := i.fun(st.getRequest(app, i.name))

					if err != nil {

						return err
					}

					fmt.Println(res)

					continue

				}

				c := NewClient(st)

				d := data{
					Name:  i.name,
					Flags: map[string]string{},
					Args:  map[string]string{},
				}

				//参数绑定
				for i2, flag := range i.flags {

					d.Flags[flag.name] = app.GetCommand(i.name).Model().Flags[i2].String()

				}

				for i2, arg := range i.args {

					d.Args[arg.name] = app.GetCommand(i.name).Model().Args[i2].String()

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

func (st *Stage) GetCancel() context.CancelFunc {

	return st.cancel
}

//------------------------------------------------------------------------------------

func (st *Stage) stop() error {

	isD, _ := PathExists(st.getDaemonName())

	if isD {

		//守护进程关闭
		sysType := runtime.GOOS

		dPid, err := read.Open(st.getDaemonName()).Read()

		if err != nil {

			return err

		}

		if sysType == `windows` {

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

		//fmt.Println("nice啊")

		return st.sendStop()

	}

	return nil
}

func (st *Stage) sendStop() error {

	c := NewClient(st)

	d := data{
		Name:  "stop",
		Flags: map[string]string{},
		Args:  map[string]string{},
	}

	str, _ := json.Marshal(d)

	_, err := c.Send(string(str))

	if err != nil {

		return err
	}

	return nil
}

func (st *Stage) getRunPidName() string {

	return st.config.RunPath + "/run.pid"
}

func (st *Stage) getSockName() string {

	return st.config.RunPath + "/temp.sock"
}

func (st *Stage) getDaemonName() string {

	return st.config.RunPath + "/daemon.pid"
}

//记录主程序pid
func (st *Stage) savePid() error {

	f, err := os.OpenFile(st.getRunPidName(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0664)

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

func (st *Stage) getOutLogName() string {

	return st.config.LogPath + "/outLog.log"
}

func (st *Stage) getOutErrName() string {

	return st.config.LogPath + "/outErr.log"
}

func (st *Stage) out(stt io.ReadCloser) {

	defer stt.Close()

	buf := make([]byte, 1024)

	f, fErr := os.OpenFile(st.getOutLogName(), os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)

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

	f, fErr := os.OpenFile(st.getOutErrName(), os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)

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

func (st *Stage) pathDeal(path string) error {

	ok, _ := PathExists(path)

	if !ok {

		mErr := os.MkdirAll(path, 0755)

		if mErr != nil {

			return mErr
		}
	}

	f, err := os.OpenFile(path+"/temp.txt", os.O_RDWR|os.O_CREATE, 0755)

	if err != nil {

		return errors.New(path + " 写入权限检查失败")
	}

	defer func() {

		f.Close()

		os.Remove(path + "/temp.txt")

	}()

	return nil
}

func (st *Stage) getItemByName(name string) (*item, error) {

	for _, i := range st.list {

		if i.name == name {

			return i, nil
		}

	}

	return nil, errors.New("no found")
}

func (st *Stage) requiredCommand() {

	startItem, _ := st.getItemByName("start")

	r := &item{
		fun:   startItem.fun,
		flags: startItem.flags,
		args:  startItem.args,
		name:  "run",
		st:    st,
		help:  "",
		hide:  true,
	}

	d := &item{
		fun:   startItem.fun,
		flags: startItem.flags,
		args:  startItem.args,
		name:  "daemon",
		st:    st,
		help:  "",
		hide:  true,
	}

	st.list = append(st.list, r, d)
}
