// graphite_load_go project main.go
package main

import (
	"bytes"
	"flag"
	"log"
	"math"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"sync"
	"time"
	"net/http"
	"io/ioutil"
)

var (
	host                      = flag.String("host", "127.0.0.1:80", "graphite-api/web host")
	cpuprofile                = flag.String("cpuprofile", "", "write cpu profile to file")
	arg_connections           = flag.Uint64("connections", 10, "Connections")
	arg_simultaniously        = flag.Uint64("simul", 10, "Simultaniously connections")
	arg_points_per_connection = flag.Uint64("points", 20, "Datapoints per connection")
	threads                   = flag.Int("threads", 2, "Threads")
	runs                      = flag.Uint64("runs", 10, "Number of runs, 0 = infinity")
	proto                     = flag.String("proto", "http", "Protocol, http/https (HTTP strongly adviced)")
    step_con                  = flag.Uint64("stepconnections", 0, "Increase number connections with this step")
    step_points               = flag.Uint64("steppoints", 0, "Increase number of points with this step")
	prefix                    = flag.String("prefix", "one_min", "Prefix for metric names")
	do_not_wait               = flag.Bool("nowait", false, "Do not wait after interation")
)

func send_data(points_per_connection uint64, n uint64) {
	var (
		i       uint64
		buf     bytes.Buffer
		params  []string
		req     string
		err     error
	)
	defer buf.Reset()
	date := time.Now().Unix()
	buf.WriteString("/render/?_dummy=" + strconv.FormatInt(date, 10))
	host_base := "&target=" + *prefix + ".perf_test.test0.metric"

	for i = 0; i < points_per_connection; i++ {
		params = []string{
			host_base,
			strconv.FormatUint(i+n, 10),
		}
		// fmt.Fprintf(conn, host_base+strconv.FormatUint(i, 10)+" "+strconv.FormatFloat(math.Sin(float64(date)+float64(i)), 'f', -1, 32)+" "+date_str+"\n")
		for _, param := range params {
			buf.WriteString(param)
		}
	}
	req = *proto + "://" + *host + buf.String()
	buf.Reset()
//	req = *proto + "://" + *host + req
	resp, err := http.Get(req)
	if err != nil {
		log.Println("Request failed:", err)
		return
	}
	defer resp.Body.Close()
	ioutil.ReadAll(resp.Body)
	/*
	res, err := ioutil.ReadAll(resp.Body)
	log.Println("URL   :", req)
	log.Println("Code  :", resp.StatusCode)
	log.Println("Header:", resp.Header)
	log.Println("Result:", string(res))
	*/
	return
}

func main() {
	var (
		i, j, cnt, connections, points_per_connection, simultaniously uint64
		wg                                                            sync.WaitGroup
		timings                                                       []uint64
	)
	flag.Parse()

	runtime.GOMAXPROCS(*threads)
	connections = *arg_connections
	points_per_connection = *arg_points_per_connection
	simultaniously = *arg_simultaniously
	if *runs > 0 {
		timings = make([]uint64, *runs)
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Println(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	cnt = 0

	log.Println("Starting...")
        for {
            for {
                    log.Println("===Iteration_", cnt + 1, "/", *runs, "===")
                    begin := time.Now().UnixNano()
					log.Println("Load    : ", connections*points_per_connection, "=",  connections, "x", points_per_connection, " metrics")

                    j = 0
                    for j < connections {
                            for i = 0; i < simultaniously; i++ {
                                    j++
                                    wg.Add(1)
                                    go func(points_per_connection, i uint64) {
                                            defer wg.Done()
                                            send_data(points_per_connection, i)
                                    }(points_per_connection, i)
                                    if j >= connections {
                                            break
                                    }
                            }
                            wg.Wait()
                    }

                    end := time.Now().UnixNano()
                    spent := uint64(end - begin)
                    log.Println("Spent   : ", strconv.FormatFloat(float64(spent/1000)/1000/1000, 'f', -1, 32), "seconds")
                    log.Println("Speed   : ", strconv.FormatFloat(float64(connections*points_per_connection)/(float64(spent/1000)/1000/1000), 'f', -1, 32), "metrics/second")

					if !*do_not_wait {
                        sleep := uint64(60 * time.Second)
                        if spent < sleep {
                                sleep -= spent
                                log.Println("Sleeping: ", strconv.FormatFloat(float64(sleep/1000)/1000/1000, 'f', -1, 32), "seconds")
                                time.Sleep(time.Duration(sleep))
                        } else {
                                log.Println("Overtime: ", strconv.FormatFloat(float64(spent-sleep/1000)/1000/1000, 'f', -1, 32), "seconds")
                        }
                    }
	
                    if *runs > 0 {
                            timings[cnt] = spent
                            cnt++
                            if cnt >= *runs {
                                    mean := float64(0.0)
                                    std := float64(0.0)
                                    cnt = 0
                                    for _, t := range timings {
                                            cnt++
                                            mean += float64(t / 1000)
                                    }
                                    mean = mean / float64(cnt)
                                    for _, t := range timings {
                                            std += math.Pow(float64(t/1000)-mean, 2.0)
                                    }
                                    std = math.Sqrt(1 / float64(cnt) * std)
                                    log.Println("Result  : ", strconv.FormatFloat(mean/1000/1000, 'f', -1, 32), "+-", strconv.FormatFloat(std/1000/1000, 'f', -1, 32), "seconds")
                                    break
                            }
                    }
            }


            if *step_con == 0 && *step_points == 0 {
                break
            } else {
                connections += *step_con
                points_per_connection += *step_points
                log.Println("Increasing load by ", *step_con, " connections and ", *step_points, " points per connection")
                log.Println("Load    : ", connections, "x", points_per_connection, "=", connections*points_per_connection, " metrics")
                cnt = 0
            }
        }
}
