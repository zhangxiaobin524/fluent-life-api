package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// getLogDir 获取日志目录路径（相对于项目根目录）
func getLogDir() string {
	// 尝试从环境变量获取项目根目录
	if projectRoot := os.Getenv("PROJECT_ROOT"); projectRoot != "" {
		return filepath.Join(projectRoot, "logs")
	}
	
	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		// 如果获取失败，使用相对路径
		return "logs"
	}
	
	// 检查是否在 cmd/server 目录下
	if filepath.Base(wd) == "server" && filepath.Base(filepath.Dir(wd)) == "cmd" {
		// 如果在 cmd/server 目录，向上两级到项目根目录
		return filepath.Join(wd, "..", "..", "logs")
	}
	
	// 检查是否在项目根目录
	if _, err := os.Stat(filepath.Join(wd, "logs")); err == nil {
		return filepath.Join(wd, "logs")
	}
	
	// 默认使用相对路径
	return "logs"
}

var (
	apiLogger     *log.Logger
	apiLogFile    *os.File
	apiLogMutex   sync.Mutex
	apiLogDate    string
)

// initAPILogger 初始化 API 日志记录器（按日期）
func initAPILogger() error {
	apiLogMutex.Lock()
	defer apiLogMutex.Unlock()

	// 获取当前日期
	currentDate := time.Now().Format("2006-01-02")

	// 如果日期没变且文件已打开，直接返回
	if apiLogFile != nil && apiLogDate == currentDate {
		return nil
	}

	// 关闭旧文件
	if apiLogFile != nil {
		apiLogFile.Close()
		apiLogFile = nil
	}

	// 创建日志目录
	logDir := getLogDir()
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// 创建按日期的日志文件
	logFileName := fmt.Sprintf("api-%s.log", currentDate)
	logPath := filepath.Join(logDir, logFileName)

	// 打开日志文件（追加模式）
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	apiLogFile = file
	apiLogDate = currentDate
	apiLogger = log.New(file, "", log.LstdFlags)

	return nil
}

// APILog 写入 API 日志（同时输出到控制台和文件，按日期）
func APILog(format string, v ...interface{}) {
	// 检查日期是否变化
	currentDate := time.Now().Format("2006-01-02")
	if apiLogDate != currentDate {
		initAPILogger()
	}

	// 确保日志文件已初始化
	if apiLogger == nil {
		if err := initAPILogger(); err != nil {
			// 如果初始化失败，只输出到控制台
			log.Printf(format, v...)
			return
		}
	}

	// 同时输出到控制台和文件
	log.Printf(format, v...)
	apiLogMutex.Lock()
	apiLogger.Printf(format, v...)
	apiLogMutex.Unlock()
}

