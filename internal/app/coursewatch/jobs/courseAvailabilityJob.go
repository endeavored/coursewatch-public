package jobs

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/endeavored/coursewatch/internal/pkg/helpers"
	"github.com/endeavored/coursewatch/internal/pkg/models"
	"github.com/endeavored/coursewatch/internal/pkg/requests"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/html"
)

type SafeCourseAvailability struct {
	models.CourseAvailability
	mu sync.Mutex
}
type SafeAvailabilityCache struct {
	a       *models.App
	Courses map[string]*SafeCourseAvailability
	sdc     *SafeDetailCache
	mu      sync.Mutex
}

const clientCount = 20

var clientPool []*fasthttp.Client = make([]*fasthttp.Client, 0)

func courseAvailabilityJob(a *models.App, sdc *SafeDetailCache) *SafeAvailabilityCache {
	return &SafeAvailabilityCache{
		Courses: make(map[string]*SafeCourseAvailability),
		sdc:     sdc,
		a:       a,
	}
}

func (sac *SafeAvailabilityCache) Run() {
	for i := 0; i < clientCount; i++ {
		clientPool = append(clientPool, &fasthttp.Client{
			TLSConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		})
	}
	for {
		classes := sac.a.Classes
		for crn, _ := range classes {
			go sac.updateCourseAvailability(crn, clientPool[rand.Intn(clientCount)])
			time.Sleep(100 * time.Millisecond)
		}
		time.Sleep(2 * time.Second)
	}
}

func (sac *SafeAvailabilityCache) updateCourseAvailability(crn string, cli *fasthttp.Client) {
	courseUrl := "https://registration.banner.gatech.edu/StudentRegistrationSsb/ssb/searchResults/getEnrollmentInfo?term=" + sac.a.Term + "&courseReferenceNumber=" + crn
	courseUrl = strings.TrimSpace(courseUrl)
	statusCode, respBody, err := requests.SimpleGetCli(cli, courseUrl)
	if err != nil {
		fmt.Println(err)
		return
	}
	if statusCode != 200 {
		fmt.Printf("course availability: %d on crn %s\n", statusCode, crn)
		fmt.Println(courseUrl)
		return
	}
	if _, ok := sac.Courses[crn]; !ok {
		if sac.mu.TryLock() {
			sac.Courses[crn] = &SafeCourseAvailability{}
			sac.mu.Unlock()
		} else {
			fmt.Println("error trying to lock sac")
			return
		}
	}
	if changed, err, isInit := parseCourse(respBody, sac.Courses[crn]); changed {
		if err != nil {
			fmt.Println(err)
		}
		changedAvail := false
		//sac.Courses[crn].WaitlistAvailable = false
		if sac.Courses[crn].WAvailable > 0 || sac.Courses[crn].WActual < sac.Courses[crn].WCapacity {
			changedAvail = !sac.Courses[crn].WaitlistAvailable
			sac.Courses[crn].WaitlistAvailable = true
		} else {
			sac.Courses[crn].WaitlistAvailable = false
		}
		//if (sac.Courses[crn].EAvailable > 0 || sac.Courses[crn].EActual < sac.Courses[crn].EMax) && sac.Courses[crn].WCapacity == 0 {
		// When waitlist drops, there will be 0 actual and capacity remains the same
		if sac.Courses[crn].WActual == 0 && (sac.Courses[crn].EAvailable > 0 || sac.Courses[crn].EActual < sac.Courses[crn].EMax) {
            fmt.Println(sac.Courses[crn].WActual)
			changedAvail = !sac.Courses[crn].EnrollmentAvailable
			sac.Courses[crn].EnrollmentAvailable = true
		} else {
			sac.Courses[crn].EnrollmentAvailable = false
		}
		if changedAvail && !isInit {
			fmt.Println(crn + " is available")
			var cd *models.CourseDetails
			if details, ok := sac.sdc.Details[crn]; ok {
				cd = &details
			} else {
				fmt.Println("Fetching new course details")
				cd, err = helpers.GetCourseDetails(sac.a.Term, crn)
				if err != nil {
					fmt.Printf("ERROR GETTING COURSE DETAILS: %v\n", err)
				}
			}

			if cd != nil {
				fmt.Println("Sending to slack")
				/*if channel, ok := sac.a.Channels[crn]; ok {
					helpers.SendToSlackDM(channel, cd, &sac.Courses[crn].CourseAvailability)
				}*/
				helpers.SendToSlack(sac.a.Webhooks, cd, &sac.Courses[crn].CourseAvailability) // TODO: Deprecate this method
			} else {
				fmt.Println("ERROR GETTING COURSE DETAILS")
			}
		}
		x, _ := json.Marshal(sac.Courses[crn])
		fmt.Println(string(x))
	}
}
func parseCourse(data []byte, sca *SafeCourseAvailability) (bool, error, bool) {
	tkn := html.NewTokenizer(bytes.NewReader(data))
	var isReadable bool = false
	var textCount int = 0
	var reachedEnd bool = false
	var changed bool = false
	var isInit bool = false
	if sca == nil {
		return false, errors.New("nil sca"), isInit
	}
	if sca.EActual == 0 && sca.EAvailable == 0 && sca.EMax == 0 && sca.WActual == 0 && sca.WAvailable == 0 && sca.WCapacity == 0 {
		isInit = true
	}
	for {
		curTokenizer := tkn.Next()
		switch {

		case curTokenizer == html.ErrorToken:
			err := tkn.Err()
			if err == io.EOF {
				reachedEnd = true
				break
			}
			return changed, err, isInit
		case curTokenizer == html.StartTagToken:
			curToken := tkn.Token()
			if len(curToken.Attr) == 0 {
				continue
			}
			if curToken.Attr[0].Key == "dir" && curToken.Attr[0].Val == "ltr" {
				isReadable = true
			}
		case curTokenizer == html.TextToken:
			if isReadable {
				curToken := tkn.Token()
				if strings.TrimSpace(curToken.Data) == "" {
					continue
				}
				isReadable = false
				textCount++
				if textCount > 6 {
					break
				}
				if textCount == 1 {
					biguint, err := strconv.ParseInt(strings.TrimSpace(curToken.Data), 10, 16)
					if err != nil {
						fmt.Println(err)
						continue
					}
					if sca.EActual == int(biguint) {
						continue
					}
					if !sca.mu.TryLock() {
						continue
					}
					changed = true
					(*sca).EActual = int(biguint)
					sca.mu.Unlock()
				} else if textCount == 2 {
					biguint, err := strconv.ParseInt(strings.TrimSpace(curToken.Data), 10, 16)
					if err != nil {
						fmt.Println(err)
						continue
					}
					if sca.EMax == int(biguint) {
						continue
					}
					if !sca.mu.TryLock() {
						continue
					}
					changed = true
					(*sca).EMax = int(biguint)
					sca.mu.Unlock()
				} else if textCount == 3 {
					biguint, err := strconv.ParseInt(strings.TrimSpace(curToken.Data), 10, 16)
					isReadable = false
					if err != nil {
						fmt.Println(err)
						continue
					}
					if sca.EAvailable == int(biguint) {
						continue
					}
					if !sca.mu.TryLock() {
						continue
					}
					changed = true
					(*sca).EAvailable = int(biguint)
					sca.mu.Unlock()
				} else if textCount == 4 {
					biguint, err := strconv.ParseInt(strings.TrimSpace(curToken.Data), 10, 16)
					if err != nil {
						fmt.Println(err)
						continue
					}
					if sca.WCapacity == int(biguint) {
						continue
					}
					if !sca.mu.TryLock() {
						continue
					}
					changed = true
					(*sca).WCapacity = int(biguint)
					sca.mu.Unlock()
				} else if textCount == 5 {
					biguint, err := strconv.ParseInt(strings.TrimSpace(curToken.Data), 10, 16)
					if err != nil {
						fmt.Println(err)
						continue
					}
					if sca.WActual == int(biguint) {
						continue
					}
					if !sca.mu.TryLock() {
						continue
					}
					changed = true
					(*sca).WActual = int(biguint)
					sca.mu.Unlock()
				} else if textCount == 6 {
					biguint, err := strconv.ParseInt(strings.TrimSpace(curToken.Data), 10, 16)
					if err != nil {
						fmt.Println(err)
						continue
					}
					if sca.WAvailable == int(biguint) {
						continue
					}
					if !sca.mu.TryLock() {
						continue
					}
					changed = true
					(*sca).WAvailable = int(biguint)
					sca.mu.Unlock()
				}
			}
		}
		if reachedEnd {
			break
		}
	}
	//fmt.Println(fmt.Sprint(cd.EActual) + " | " + fmt.Sprint(cd.EMax) + " | " + fmt.Sprint(cd.EAvailable))
	return changed, nil, isInit
}
