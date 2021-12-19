package main

import (
	"flag"
	"github.com/ainilili/tdsql-competition/database"
	"github.com/ainilili/tdsql-competition/log"
	"github.com/ainilili/tdsql-competition/pprof"
	"github.com/ainilili/tdsql-competition/table"
	"sync"
	"time"
)

var dataPath *string
var dstIP *string
var dstPort *int
var dstUser *string
var dstPassword *string

//  example of parameter parse, the final binary should be able to accept specified parameters as requested
//
//  usage example:
//      ./run --data_path /tmp/data --dst_ip 127.0.0.1 --dst_port 3306 --dst_user root --dst_password 123456789
//
//  you can test this example by:
//  go run main.go --data_path /tmp/data --dst_ip 127.0.0.1 --dst_port 3306 --dst_user root --dst_password 123456789
func init() {
	dataPath = flag.String("data_path", "D:\\workspace-tencent\\data", "dir path of source data")
	dstIP = flag.String("dst_ip", "tdsqlshard-n756r9nq.sql.tencentcdb.com", "ip of dst database address")
	dstPort = flag.Int("dst_port", 113, "port of dst database address")
	dstUser = flag.String("dst_user", "nico", "user name of dst database")
	dstPassword = flag.String("dst_password", "Niconico2021@", "password of dst database")

	flag.Parse()
}

func main() {
	start := time.Now().UnixNano()
	_main()
	log.Infof("time-consuming %dms", (time.Now().UnixNano()-start)/1e6)
}

func _main() {
	go func() {
		_ = pprof.StartPprofServer()
	}()
	db, err := database.New(*dstIP, *dstPort, *dstUser, *dstPassword)
	if err != nil {
		log.Panic(err)
	}
	tables, err := table.ParseTables(db, *dataPath)
	if err != nil {
		log.Panic(err)
	}
	initLimit := make(chan bool, 4)
	syncLimit := make(chan bool, 4)
	for i := 0; i < cap(syncLimit); i++ {
		syncLimit <- true
		initLimit <- true
	}
	wg := sync.WaitGroup{}
	wg.Add(len(tables))
	go func() {
		for i := 0; i < len(tables); i++ {
			_ = <-initLimit
			index := i
			go func() {
				rows, err := tables[index].Init()
				initLimit <- true
				if err != nil {
					return
				}
				_ = <-syncLimit
				defer func() {
					syncLimit <- true
					wg.Add(-1)
				}()
				err = tables[index].Sync(rows)
				if err != nil {
					log.Error(err)
				}
			}()
		}
	}()
	wg.Wait()
}
