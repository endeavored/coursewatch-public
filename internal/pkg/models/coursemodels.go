package models

type CourseDetails struct {
	CRN           string `json:"crn"`
	SectionNumber string `json:"sectionNumber"`
	Subject       string `json:"subject"`
	CourseNumber  string `json:"courseNumber"`
	CourseTitle   string `json:"courseTitle"`
}

type CourseAvailability struct {
	EActual             int  `json:"Enrollment Actual"`
	EMax                int  `json:"Enrollment Max"`
	EAvailable          int  `json:"Enrollment Seats Available"`
	WCapacity           int  `json:"Waitlist Capacity"`
	WActual             int  `json:"Waitlist Actual"`
	WAvailable          int  `json:"Waitlist Seats Available"`
	EnrollmentAvailable bool `json:"Enrollment Available"`
	WaitlistAvailable   bool `json:"Waitlist Available"`
}
