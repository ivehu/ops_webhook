package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

// 初始化函数，在程序启动时执行
func init() {
	// 设置程序的最大并发数为CPU核数
	runtime.GOMAXPROCS(runtime.NumCPU())
	// 设置日志的输出格式，包括日期、时间、文件名和行号
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

type ResponseMsg struct {
	Msg string `json:"msg"`
}

// 获取命令行参数
func getArgs() (int64, string, []string) {
	// 初始化 viper 以读取配置文件
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	// 读取配置文件
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	// 从配置文件中获取端口号
	portstr := viper.GetString("port")
	if portstr == "" {
		portstr = "10020" // 默认端口号
	}

	// 从配置文件中获取认证字符串
	authstr := viper.GetString("authstr")
	if authstr == "" {
		authstr = "ddu9a1XR56ZExcjg" // 默认认证字符串
	}

	// 将端口号转换为 int64 类型
	port, err := strconv.ParseInt(portstr, 10, 64)
	if err != nil {
		log.Fatalln("cannot parse port:", portstr)
	}

	// 从配置文件中获取允许执行的命令列表
	commands := viper.GetStringSlice("commands")

	return port, authstr, commands
}

// authMiddleware 函数用于验证请求的授权信息
func authMiddleware(authstr string, next http.Handler) http.Handler {
	// 返回一个 http.HandlerFunc 类型的函数，该函数接收一个 http.ResponseWriter 和一个 *http.Request 作为参数
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 从请求头中获取 Authorization 字段
		auth := r.Header.Get("Authorization")
		// 如果 Authorization 字段为空或者不等于传入的 authstr，则返回 Unauthorized 错误
		if strings.TrimSpace(auth) == "" || strings.TrimPrefix(auth, "Bearer ") != authstr {
			log.Printf("Unauthorized request from %s: %s", r.RemoteAddr, r.URL.Path)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		// 否则，调用 next 的 ServeHTTP 方法，继续处理请求
		log.Printf("Authorized request from %s: %s", r.RemoteAddr, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func main() {
	port, authstr, commands := getArgs() // Update the variable assignment to receive three values
	startHttp(port, authstr, commands)
}

func startHttp(port int64, authstr string, commands []string) {
	r := mux.NewRouter()

	r.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received GET request from %s: %s", r.RemoteAddr, r.URL.Path)
		fmt.Fprintf(w, "pong")
		log.Printf("Sent response to %s: pong", r.RemoteAddr)
	}).Methods("GET")

	r.HandleFunc("/run", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received POST request from %s: %s", r.RemoteAddr, r.URL.Path)
		if authstr == "" {
			log.Printf("Not supported. authstr is empty for request from %s: %s", r.RemoteAddr, r.URL.Path)
			http.Error(w, "not supported. use ./httpd <port> <auth-string> to enable", 200)
			return
		}

		if r.Body == nil {
			log.Printf("Bad request. Body is nil for request from %s: %s", r.RemoteAddr, r.URL.Path)
			http.Error(w, "body is nil", http.StatusBadRequest)
			return
		}

		auth := r.Header.Get("Authorization")
		if strings.TrimSpace(auth) == "" {
			log.Printf("Bad request. Authorization is blank for request from %s: %s", r.RemoteAddr, r.URL.Path)
			http.Error(w, "Authorization is blank", http.StatusBadRequest)
			return
		}

		auth = strings.TrimPrefix(auth, "Bearer ")
		if auth != authstr {
			log.Printf("Forbidden. Authorization invalid for request from %s: %s", r.RemoteAddr, r.URL.Path)
			http.Error(w, "Authorization invalid", http.StatusForbidden)
			return
		}

		defer r.Body.Close()
		bs, _ := io.ReadAll(r.Body)
		sh := string(bs)

		// 检查命令是否允许执行
		allowed := false
		for _, cmdPattern := range commands {
			if customMatch(cmdPattern, sh) {
				allowed = true
				break
			}
		}

		if !allowed {
			log.Printf("Command not allowed: %s from %s", sh, r.RemoteAddr)
			http.Error(w, fmt.Sprintf("Command `%s` is not allowed", sh), http.StatusForbidden)
			return
		}

		cmd := exec.Command("sh", "-c", sh)
		cmd.Env = append(os.Environ(), "TESTAAA=your_value")
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Execution failed for request from %s: %s. Error: %v", r.RemoteAddr, r.URL.Path, err)
			http.Error(w, fmt.Sprintf("exec `%s` fail: %v", sh, err), 200)
			return
		}
		resp := ResponseMsg{
			Msg: string(output),
		}
		jsonBytes, err := json.Marshal(resp)
		if err != nil {
			log.Printf("JSON marshaling failed for request from %s: %s. Error: %v", r.RemoteAddr, r.URL.Path, err)
			w.Write(output)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBytes)
		log.Printf("Sent response to %s: %s", r.RemoteAddr, string(jsonBytes))
	}).Methods("POST")

	// 静态文件服务路由
	//r.PathPrefix("/").Handler(http.FileServer(http.Dir("./")))

	n := negroni.New()
	n.Use(negroni.HandlerFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		authMiddleware(authstr, next).ServeHTTP(w, r)
	}))
	n.UseHandler(r)

	log.Println("listening http on", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), n))
}

// customMatch 自定义命令匹配函数
func customMatch(pattern, command string) bool {
	patternParts := strings.Fields(pattern)
	commandParts := strings.Fields(command)

	if len(patternParts) == 0 || len(commandParts) == 0 {
		return false
	}

	// 检查命令名是否匹配
	if patternParts[0] != commandParts[0] {
		return false
	}

	// 如果模式只有命令名，认为所有该命令都允许
	if len(patternParts) == 1 {
		return true
	}

	// 处理模式中有 * 的情况
	if patternParts[1] == "*" {
		return true
	}

	// 精确匹配参数
	return strings.Join(patternParts[1:], " ") == strings.Join(commandParts[1:], " ")
}
