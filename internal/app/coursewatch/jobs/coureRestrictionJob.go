package jobs

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/endeavored/coursewatch/internal/pkg/models"
	"github.com/endeavored/coursewatch/internal/pkg/requests"
	"golang.org/x/net/html"
)

//https://registration.banner.gatech.edu/StudentRegistrationSsb/ssb/searchResults/getRestrictions?term=202308&courseReferenceNumber=80673

type SafeCourseRestrictions struct {
	Restrictions map[string]bool
	mu           sync.Mutex
}
type SafeRestrictionCache struct {
	a       *models.App
	Courses map[string]*SafeCourseRestrictions
	mu      sync.Mutex
}

func courseRestrictionJob(a *models.App) *SafeRestrictionCache {
	return &SafeRestrictionCache{
		a:       a,
		Courses: make(map[string]*SafeCourseRestrictions),
		mu:      sync.Mutex{},
	}
}

func (src *SafeRestrictionCache) Run() {
	for {
		classes := src.a.Classes
		for crn, _ := range classes {
			_ = crn
			time.Sleep(100 * time.Millisecond)
		}
		time.Sleep(1 * time.Minute)
	}
}

func (src *SafeRestrictionCache) updateCourseRestrictions(crn string) {
	courseUrl := "https://registration.banner.gatech.edu/StudentRegistrationSsb/ssb/searchResults/getRestrictions?term=" + src.a.Term + "&courseReferenceNumber=" + crn
	courseUrl = strings.TrimSpace(courseUrl)
	statusCode, respBody, err := requests.SimpleGet(courseUrl)
	_ = respBody
	if err != nil {
		fmt.Println(err)
		return
	}
	if statusCode != 200 {
		fmt.Printf("course restrictions: %d on crn %s\n", statusCode, crn)
		fmt.Println(courseUrl)
		return
	}
	if _, ok := src.Courses[crn]; !ok {
		if src.mu.TryLock() {

		} else {
			fmt.Println("error trying to lock src")
			return
		}
	}
}

func parseCourseRestrictions(data []byte) {
	tkn := html.NewTokenizer(bytes.NewReader(data))
	var reachedEnd bool = false
	for {
		curTokenizer := tkn.Next()
		switch {
		case curTokenizer == html.ErrorToken:
			err := tkn.Err()
			if err == io.EOF {
				reachedEnd = true
			}
		}
	}
	_ = reachedEnd
}
