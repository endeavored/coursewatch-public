package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/endeavored/coursewatch/internal/app/coursewatch/jobs"
	"github.com/endeavored/coursewatch/internal/pkg/heroku"
	"github.com/endeavored/coursewatch/internal/pkg/models"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/valyala/fasthttp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type WrappedClassData struct {
	Id        string          `json:"_id"`
	ClassData map[string]bool `json:"classData"`
}

var (
	mongoClient           *mongo.Client
	mongoConnectionString string = ""
)

func Start() {
	err := godotenv.Load()

	if err != nil {
		fmt.Println(err)
	}
	//This is where main app starting will be handled, all logic here
	//Init tasks go here
	a := &models.App{
		Term:     "202508",
		Ctx:      context.Background(),
		Classes:  make(map[string]bool),
		Webhooks: make([]string, 0),
	}
	savedWebhook := os.Getenv("SLACK_WEBHOOK")
	if savedWebhook != "" {
		a.Webhooks = append(a.Webhooks, savedWebhook)
	}
	socketUrl, err := getSocketUrl()
	if err != nil {
		log.Fatal(err)
	}
	go connectToWebsocket(a, socketUrl)

	jobs.Start(a)
	port := os.Getenv("PORT")
	mongoConnectionString = os.Getenv("mongoConnectionString")
	if mongoConnectionString != "" {
		serverAPI := options.ServerAPI(options.ServerAPIVersion1)
		opts := options.Client().ApplyURI(mongoConnectionString).SetServerAPIOptions(serverAPI)
		client, err := mongo.Connect(context.TODO(), opts)
		if err != nil {
			panic(err)
		}
		defer func() {
			if err = client.Disconnect(context.TODO()); err != nil {
				panic(err)
			}
		}()

		res := client.Database("monitor-data").Collection("classes").FindOne(context.TODO(), bson.D{})
		if res == nil {
			fmt.Println("couldnt find class data")
		}

		var wClassData *WrappedClassData = &WrappedClassData{}
		if err := res.Decode(wClassData); err != nil {
			log.Fatal(err)
		}

		for k, v := range wClassData.ClassData {
			k = strings.TrimSpace(k)
			a.Classes[k] = v
			fmt.Printf("Added class %s from database\n", k)
		}
		mongoClient = client
	}

	http.HandleFunc("/", webHandler)
	go heroku.StartHeartbeat(port)
	log.Panic(http.ListenAndServe(":"+port, nil))

}

func webHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "ok")
}

func getSocketUrl() (string, error) {
	headers := make(map[string]string)
	headers["Authorization"] = "Bearer " + os.Getenv("SLACK_SOCKET_TOKEN")
	db, err := postRequest("https://slack.com/api/apps.connections.open", nil, headers)
	if err != nil {
		time.Sleep(time.Second * 3)
		return getSocketUrl()
	}
	var result map[string]interface{}
	json.Unmarshal(db, &result)
	socket_url := fmt.Sprint(result["url"])
	return socket_url, nil
}

func connectToWebsocket(a *models.App, socketUrl string) {
	conn, resp, err := websocket.DefaultDialer.Dial(socketUrl, nil)
	if err != nil {
		//log.Fatalf("error: %s\nerror code: %d\n", err, resp.StatusCode)
		fmt.Printf("error: %v\n", err)
		if resp != nil {
			fmt.Printf("error code: %d\n", resp.StatusCode)
		}
		time.Sleep(10 * time.Second)
		connectToWebsocket(a, socketUrl)
		if conn != nil {
			conn.Close()
		}
		return
	}
	defer conn.Close()
	receiveHandler(a, conn)
	fmt.Println("socket closed unexpectedly")
	time.Sleep(5 * time.Second)
	fmt.Println("attempting reconnect to socket")
	socketUrl, err = getSocketUrl()
	if err != nil {
		log.Fatal(err)
	}
	connectToWebsocket(a, socketUrl)
}

func receiveHandler(a *models.App, connection *websocket.Conn) {
	for {
		_, msg, err := connection.ReadMessage()
		if err != nil {
			log.Println("Error in receive:", err)
			return
		}
		//log.Printf("Received: %s\n", msg)
		var receivedData models.SlackSocketData
		err = json.Unmarshal([]byte(msg), &receivedData)
		if err == nil {
			command := receivedData.Payload.Command
			if command != "" {
				text := receivedData.Payload.Text
				username := receivedData.Payload.UserName
				rtCommand := "added"
				if command == "/remove-crn" {
					rtCommand = "removed"
				}
				numericRegex := regexp.MustCompile("[^0-9]+")
				text = numericRegex.ReplaceAllString(text, "")
				if text == "" {
					var sendData models.SlackSocketData
					sendData.EnvelopeId = receivedData.EnvelopeId
					sendData.Payload.Text = "CRN must be valid"
					connection.WriteJSON(sendData)
					return
				}
				fmt.Println(text)
				text = strings.TrimSpace(text)
				if rtCommand == "added" {
					crn := text
					a.Classes[crn] = true
				} else if rtCommand == "removed" {
					crn := text
					delete(a.Classes, crn)
				}
				if mongoClient != nil {
					serverAPI := options.ServerAPI(options.ServerAPIVersion1)
					opts := options.Client().ApplyURI(mongoConnectionString).SetServerAPIOptions(serverAPI)
					client, err := mongo.Connect(context.TODO(), opts)
					if err != nil {
						fmt.Println(err)
					} else {
						client.Database("monitor-data").Collection("classes").FindOneAndReplace(context.TODO(), bson.D{}, WrappedClassData{
							ClassData: a.Classes,
						})
						fmt.Println("replaced db")
					}
				}
				fmt.Println(command + " " + text + " sent from " + username)
				var sendData models.SlackSocketData
				sendData.EnvelopeId = receivedData.EnvelopeId
				sendData.Payload.Text = "CRN " + text + " has been " + rtCommand
				fmt.Println(sendData)
				err := connection.WriteJSON(sendData)
				fmt.Println(err)
			}
		}
	}
}

func postRequest(reqUri string, data []byte, headers ...map[string]string) ([]byte, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(reqUri)
	req.Header.SetMethod("POST")
	for i := 0; i < len(headers); i++ {
		curHeaderList := headers[i]
		for key, value := range curHeaderList {
			if strings.ToLower(key) == "content-type" {
				req.Header.SetContentType(value)
			} else {
				req.Header.Add(key, value)
			}
		}
	}
	req.SetBody(data)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := fasthttp.Do(req, resp)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		fmt.Printf("Request returned status: %d\n", resp.StatusCode())
		fmt.Println(string(resp.Body()))
		return nil, fmt.Errorf("status code: %d", resp.StatusCode())
	}

	contentEncoding := resp.Header.Peek("Content-Encoding")
	var respBody []byte
	if bytes.EqualFold(contentEncoding, []byte("gzip")) {
		respBody, _ = resp.BodyGunzip()
	} else {
		respBody = resp.Body()
	}
	return respBody, nil
}
