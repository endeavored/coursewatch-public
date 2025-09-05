package heroku

import (
	"fmt"
	"time"

	"github.com/valyala/fasthttp"
)

func StartHeartbeat(port string) {
	time.Sleep(5 * time.Second)
	for {
		makeHeartbeat(port)
		time.Sleep(15 * time.Minute)
	}
}

func makeHeartbeat(port string) {
	statusCode, bod, err := fasthttp.Get(nil, "https://gt-monitors-2ed30aa04f24.herokuapp.com/")
	if err != nil {
		fmt.Printf("heartbeat received error %v\n", err)
		return
	}
	if statusCode != 200 {
		fmt.Printf("heartbeat received status %d\n", statusCode)
		return
	}
	fmt.Println("heartbeat got " + string(bod))
}
