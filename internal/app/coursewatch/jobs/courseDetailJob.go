package jobs

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/endeavored/coursewatch/internal/pkg/helpers"
	"github.com/endeavored/coursewatch/internal/pkg/models"
)

type SafeDetailCache struct {
	a       *models.App
	Details map[string]models.CourseDetails
	mu      sync.Mutex
}

func courseDetailJob(a *models.App) *SafeDetailCache {
	return &SafeDetailCache{
		a:       a,
		Details: make(map[string]models.CourseDetails),
		mu:      sync.Mutex{},
	}
}

func (sdc *SafeDetailCache) Run() {
	for {
		classes := sdc.a.Classes
		for crn, _ := range classes {
			go sdc.updateCourseDetails(crn)
			time.Sleep(100 * time.Millisecond)
		}
		time.Sleep(20 * time.Minute)
	}
}

func (sdc *SafeDetailCache) updateCourseDetails(crn string) {
	cd, err := helpers.GetCourseDetails(sdc.a.Term, crn)
	if err != nil {
		fmt.Printf("ERROR UPDATING COURSE DETAILS: %v\n", err)
		return
	}
	if cd == nil {
		fmt.Println("NIL COURSE DETAILS")
		return
	}
	a, _ := json.Marshal(cd)
	fmt.Println(string(a))
	if sdc.mu.TryLock() {
		sdc.Details[crn] = *cd
		sdc.mu.Unlock()
	}
}
