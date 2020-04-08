package main

import (
	"encoding/json"
	"fmt"
	"github.com/mediocregopher/radix/v3"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Defaults
const DefaultFlushInterval = (150 * time.Microsecond) //seconds
const DefaultFlushSize = 10                           //records
const DefaultConnNetwork = "tcp"                      //tcp
const DefaultConnPoolScaleFactor = 1                  //initial PoolSize x ScaleFactor
const DefaultConnPoolSize = 5
const DefaulConnPingInterval = (90 * time.Second)
const DefaultConnHost = "127.0.0.1"
const DefaultConnPort = 6379
const DefaultRunOnce = false
const DefaultStreamOut = false
const DefaultStreamName = "raggs"

type Doc struct {
	Key      string                  `json:"key"`
	Datatype string                  `json:"datatype"`
	Date     string                  `json:"date"`
	Data     *map[string]interface{} `json:"data"`
}

type Service struct {
	connPool   *radix.Pool
	streamOut  bool
	streamName string
}

func generateKey(datatype []string, delimiter string) string {
	key := strings.Join(datatype, delimiter)
	return key
}

func (svc *Service) Init() error {
	var err error
	var connHost string
	var connNetwork string
	var connPort int
	var connPoolSize int
	var connScaleFactor int
	var connPingInterval time.Duration
	var flushSize int
	var flushInterval time.Duration
	var runOnce bool

	// Disable for now
	//if os.Getenv("REDIS_HOST") == "" {
	connNetwork = DefaultConnNetwork
	//} else {
	//	redisHost, err := os.Getenv("REDIS_HOST")
	//	if err != nil {
	//		return err
	//	}
	//}

	fmt.Println("Load vars...")
	// Check for REDIS_HOST.
	if os.Getenv("REDIS_HOST") == "" {
		connHost = DefaultConnHost
	} else {
		connHost = os.Getenv("REDIS_HOST")
	}

	// Check for REDIS_PORT.
	if os.Getenv("REDIS_PORT") == "" {
		connPort = DefaultConnPort
	} else {
		connPort, err = strconv.Atoi(os.Getenv("REDIS_PORT"))
		if err != nil {
			return err
		}
	}

	connAddr := connHost + ":" + strconv.Itoa(connPort)

	// Check if for REDIS_POOL_SIZE.
	if os.Getenv("REDIS_POOL_SIZE") == "" {
		connPoolSize = DefaultConnPoolSize
	} else {
		connPoolSize, err = strconv.Atoi(os.Getenv("REDIS_POOL_SIZE"))
		if err != nil {
			return err
		}
	}

	// Check if for REDIS_POOL_SCALE_FACTOR.
	if os.Getenv("REDIS_POOL_SCALE_FACTOR") == "" {
		connScaleFactor = DefaultConnPoolScaleFactor
	} else {
		connScaleFactor, err = strconv.Atoi(os.Getenv("REDIS_POOL_SCALE_FACTOR"))
		if err != nil {
			return err
		}
	}

	// Check for REDIS_PING_INTERVAL
	if os.Getenv("REDIS_PING_INTERVAL") == "" {
		connPingInterval = DefaulConnPingInterval
	} else {
		connPingInterval, err = time.ParseDuration(os.Getenv("REDIS_PING_INTERVAL"))
		if err != nil {
			return err
		}
	}

	// Check for FLUSH_SIZE
	if os.Getenv("FLUSH_SIZE") == "" {
		flushSize = DefaultFlushSize
	} else {
		flushSize, err = strconv.Atoi(os.Getenv("FLUSH_SIZE"))
		if err != nil {
			return err
		}
	}

	//Check for FLUSH_INTERVAL
	if os.Getenv("FLUSH_INTERVAL") == "" {
		flushInterval = DefaultFlushInterval
	} else {
		flushInterval, err = time.ParseDuration(os.Getenv("FLUSH_INTERVAL"))
		if err != nil {
			return err
		}
	}

	// Check for the RUN_ONCE
	if os.Getenv("RUN_ONCE") == "" {
		runOnce = DefaultRunOnce
	} else {
		runOnce, err = strconv.ParseBool(os.Getenv("RUN_ONCE"))
		if err != nil {
			return err
		}
	}

	// Check for the REDIS_STREAM_OUT
	if os.Getenv("REDIS_STREAM_OUT") == "" {
		svc.streamOut = DefaultStreamOut
	} else {
		svc.streamOut, err = strconv.ParseBool(os.Getenv("REDIS_STREAM_OUT"))
		if err != nil {
			return err
		}
	}

	// Check for REDIS_HOST.
	if os.Getenv("REDIS_STREAM_NAME") == "" {
		svc.streamName = DefaultStreamName
	} else {
		svc.streamName = os.Getenv("REDIS_STREAM_NAME")
	}

	fmt.Println("Connecting to redis host: " + connAddr)
	// Setup the elastic client.
	connPool, err := radix.NewPool(
		connNetwork,
		connAddr,
		connPoolSize,
		radix.PoolOnFullBuffer((connPoolSize*connScaleFactor), 5*time.Second),
		radix.PoolPingInterval(connPingInterval),
		radix.PoolPipelineWindow(flushInterval, flushSize),
	)
	if err != nil {
		log.Fatalf("Could not connect to redis %v", err)
	}

	svc.connPool = connPool

	// Setup http server.
	mux := http.NewServeMux()

	// Setup the http handle for incoming logs.
	mux.HandleFunc("/", svc.handler)
	mux.HandleFunc("/bulk", svc.bulk)
	mux.HandleFunc("/ping", svc.ping)

	// Run once mode for unit tests.
	if runOnce {
		return nil
	}

	return http.ListenAndServe(":3000", mux)
}

// Private

func (svc *Service) ping(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	w.WriteHeader(http.StatusOK)
}

func (svc *Service) bulk(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method == "POST" {
		// Decode the incoming payload that will be placed into the bulker.
		decoder := json.NewDecoder(r.Body)

		// read open bracket
		t, err := decoder.Token()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			//log.Fatalf("Could not parse payload: v%", err)
			fmt.Printf("%T: %v\n", t, t)
		}

		// while the array contains values
		for decoder.More() {
			var m Doc
			// decode an array value (Message)
			err := decoder.Decode(&m)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				//log.Fatalf("Could not parse bulk item: v%", err)
				fmt.Println(err)
			}

			m.Key = generateKey([]string{m.Datatype, m.Key}, ":")

			//implicit pipelining should convert flush multiple requests in the
			err = svc.connPool.Do(radix.FlatCmd(nil, "HMSET", m.Key, m.Data))
			if err != nil {
				panic(err)
			}

			if svc.streamOut {
				var responseVal map[string]string
				err := svc.connPool.Do(radix.FlatCmd(&responseVal, "HGETALL", m.Key))
				if err != nil {
					panic(err)
				} else {
					err := svc.connPool.Do(radix.FlatCmd(nil, "XADD", svc.streamName, "*", responseVal))
					if err != nil {
						panic(err)
					}
				}
			}

		}

		// read closing bracket
		t, err = decoder.Token()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			//log.Fatalf("Could not parse bulk item: v%", err)
			fmt.Printf("%T: %v\n", t, t)
		}

		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (svc *Service) handler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method == "POST" {
		// Decode the incoming payload that will be placed into the bulker.
		decoder := json.NewDecoder(r.Body)

		var doc Doc
		err := decoder.Decode(&doc)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Break up the path to get the type and key.
		path := strings.Split(r.URL.Path, "/")

		// Must include a path which will be the index and type you are targeting.
		if len(path) != 3 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		t := time.Now()

		doc.Date = t.Format(time.RFC3339)
		doc.Datatype = path[1]
		doc.Key = generateKey(path[1:2], ":")

		fmt.Println(doc.Data)

		//r := elastic.NewBulkIndexRequest().Index(fmt.Sprintf(`%s-%s`, path[1], t.Format("2006-01-02"))).Type(path[2]).Doc(doc)
		err = svc.connPool.Do(radix.FlatCmd(nil, "HMSET", doc.Key, doc.Data))
		if err != nil {
			panic(err)
		}

		if svc.streamOut {
			var responseVal map[string]string
			err := svc.connPool.Do(radix.FlatCmd(&responseVal, "HGETALL", doc.Key))
			if err != nil {
				panic(err)
			} else {
				err := svc.connPool.Do(radix.FlatCmd(nil, "XADD", svc.streamName, "*", responseVal))
				if err != nil {
					panic(err)
				}
			}
		}
		w.WriteHeader(http.StatusNoContent)
		return
	} else if r.Method == "GET" {
		var doc Doc

		// Break up the path to get the type and key.
		path := strings.Split(r.URL.Path, "/")

		// Must include a path which will be the index and type you are targeting.
		if len(path) != 3 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		doc.Key = path[1] + ":" + path[2]

		//r := elastic.NewBulkIndexRequest().Index(fmt.Sprintf(`%s-%s`, path[1], t.Format("2006-01-02"))).Type(path[2]).Doc(doc)
		var responseVal map[string]string
		err := svc.connPool.Do(radix.FlatCmd(&responseVal, "HGETALL", doc.Key))
		if err != nil {
			panic(err)
		}

		js, err := json.Marshal(responseVal)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}
