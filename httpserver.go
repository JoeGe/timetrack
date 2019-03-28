package main

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"
)

var (
	logPath = flag.String("log-path", "./", "logging path")
	port    = flag.Uint("port", 80, "server port")
)

type storage interface {
	get(key string) (value string, err error)
	store(key string, value string) error
}

type simpleMemStorage struct {
	storage map[string]string
}

func (s *simpleMemStorage) get(key string) (value string, err error) {
	return "", nil
}

func (s *simpleMemStorage) store(key string, value string) error {
	return nil
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Parse()

	dbFile := *logPath + "db"

	requestLogFile := *logPath + "request.log"
	requestLogFd, err := os.OpenFile(requestLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		log.Fatal(err)
	}

	stamps = make(stampSet)

	dbBytes, err := ioutil.ReadFile(dbFile)
	if len(dbBytes) != 0 {
		if err != nil {
			log.Fatalln("read from db file failed.", err)
		}
		err = json.Unmarshal(dbBytes, &stamps)
		if err != nil {
			log.Fatalln("read former stamps failed.", err)
		}
	}
	defer func() {
		stampsBytes, _ := json.Marshal(stamps)
		_ = ioutil.WriteFile(dbFile, stampsBytes, 0644)
	}()

	ttServer := echo.New()
	ttServer.HideBanner = true
	ttServer.Use(middleware.Recover())
	ttServer.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{Output: requestLogFd}))
	ttServer.GET("/stamp", stampHandler)
	ttServer.GET("/list", listHandler)
	go func() {
		err := ttServer.Start(":" + strconv.Itoa(int(*port)))
		if err != nil {
			ttServer.Logger.Fatal(err)
		}
	}()

	// gracefully shutdown ttServer
	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := ttServer.Shutdown(ctx); err != nil {
		ttServer.Logger.Fatal(err)
	}
}

func stampHandler(c echo.Context) error {
	begin := c.QueryParam("begin")
	finish := c.QueryParam("finish")

	cstSh, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		c.Echo().Logger.Fatal("get time location failed, %s", err)
		return c.JSON(http.StatusInternalServerError, "time location error")
	}
	now := time.Now().In(cstSh)
	d := now.Format("2006-01-02")
	t := now.Format("15:04")

	sEnt, ok := stamps[day(d)]
	if !ok {
		stamps[day(d)] = make(stampEntry)
		sEnt, _ = stamps[day(d)]
	}
	sEnt[timeofday(t)] = *&StampContent{
		Begin:  begin,
		Finish: finish,
	}

	return c.JSONPretty(http.StatusOK, stamps[day(d)], "  ")
}

func listHandler(c echo.Context) error {
	d := c.QueryParam("day")

	sEnt, ok := stamps[day(d)]
	if !ok {
		return c.JSONPretty(http.StatusNotFound, "stamps of day not found", "  ")
	}
	return c.JSONPretty(http.StatusOK, sEnt, "  ")
}

type StampContent struct {
	Begin  string `json:"start:"`
	Finish string `json:"finish"`
}

type timeofday string
type stampEntry map[timeofday]StampContent

type day string
type stampSet map[day]stampEntry

var stamps stampSet
