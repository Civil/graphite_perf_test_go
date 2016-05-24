// graphite_load_go project main.go
package main

import (
	"bytes"
	"flag"
	"log"
	"math"
	"math/rand"
	"net"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"sync"
	"time"
)

var (
	host                      = flag.String("host", "127.0.0.1:2003", "graphite host")
	cpuprofile                = flag.String("cpuprofile", "", "write cpu profile to file")
	arg_connections           = flag.Uint64("connections", 10000, "Connections")
	arg_simultaniously        = flag.Uint64("simul", 1000, "Simultaniously connections")
	arg_points_per_connection = flag.Uint64("points", 1000, "Datapoints per connection")
	threads                   = flag.Int("threads", 2, "Threads")
	runs                      = flag.Uint64("runs", 0, "Number of runs, 0 = infinity")
	proto                     = flag.String("proto", "tcp", "Protocol, tcp/udp")
	step_con                  = flag.Uint64("stepconnections", 0, "Increase number connections with this step")
	step_points               = flag.Uint64("steppoints", 0, "Increase number of points with this step")
	do_not_wait               = flag.Bool("nowait", false, "Do not wait after interation")
)

func send_data(conn net.Conn, points_per_connection uint64, n uint64) {
	var (
		i      uint64
		buf    bytes.Buffer
		params []string
		num    int
		err    error
	)
	defer conn.Close()
	date := time.Now().Unix()
	host_base := "one_min.perf_test.test" + strconv.FormatUint(n, 10) + ".metric"
	end_str := " " + strconv.FormatFloat(rand.Float64(), 'f', -1, 32) + " " + strconv.FormatInt(date, 10) + "\n"

	for i = 0; i < points_per_connection; i++ {
		params = []string{
			host_base,
			strconv.FormatUint(i, 10),
			end_str,
		}
		// fmt.Fprintf(conn, host_base+strconv.FormatUint(i, 10)+" "+strconv.FormatFloat(math.Sin(float64(date)+float64(i)), 'f', -1, 32)+" "+date_str+"\n")
		for _, param := range params {
			buf.WriteString(param)
		}
	}

	num, err = conn.Write(buf.Bytes())
	if err != nil {
		log.Println("Written: %i, error: %s", num, err.Error())
	}
	buf.Reset()

	return
}

func main() {
	var (
		i, j, cnt, connections, points_per_connection, simultaniously uint64
		wg                                                            sync.WaitGroup
		conn                                                          net.Conn
		err                                                           error
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
	fails := uint64(0)

	log.Println("Starting...")
	for {
		for {
			log.Println("===Iteration", cnt, "/", *runs, "===")
			begin := time.Now().UnixNano()
			log.Println("Load    : ", connections*points_per_connection, "=", connections, "x", points_per_connection, " metrics")

			j = 0
			fails = 0
			for j < connections {
				for i = 0; i < simultaniously; i++ {
					j++
					conn, err = net.DialTimeout(*proto, *host, 150*time.Millisecond)
					if err != nil {
						fails++
						log.Println("GoRoutine ", i, ", error: ", err)
						continue
					}

					wg.Add(1)
					go func(conn net.Conn, points_per_connection, i uint64) {
						defer wg.Done()
						send_data(conn, points_per_connection, i)
					}(conn, points_per_connection, i)
					if j >= connections {
						break
					}
				}
				wg.Wait()
			}

			end := time.Now().UnixNano()
			spent := uint64(end - begin)
			log.Println("Spent   : ", strconv.FormatFloat(float64(spent/1000)/1000/1000, 'f', -1, 32), "seconds")
			log.Println("Speed   : ", strconv.FormatFloat(float64((connections-fails)*points_per_connection)/(float64(spent/1000)/1000/1000), 'f', -1, 32), "metrics/second")
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
