package helpers

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"strings"

	"github.com/endeavored/coursewatch/internal/pkg/models"
	"github.com/endeavored/coursewatch/internal/pkg/requests"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/html"
)

var cdCli *fasthttp.Client = &fasthttp.Client{
	TLSConfig: &tls.Config{
		InsecureSkipVerify: true,
	},
}

func GetCourseDetails(term string, crn string) (*models.CourseDetails, error) {
	statusCode, respBody, err := requests.SimpleGetCli(cdCli, "https://registration.banner.gatech.edu/StudentRegistrationSsb/ssb/searchResults/getClassDetails?term="+term+"&courseReferenceNumber="+crn)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	if statusCode != 200 {
		fmt.Println(statusCode)
		return nil, fmt.Errorf("course got status: %d on crn %s", statusCode, crn)
	}

	details, err := parseCourse(respBody)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	details.CRN = crn

	return details, nil
}

func parseCourse(body []byte) (*models.CourseDetails, error) {
	tkn := html.NewTokenizer(bytes.NewReader(body))
	var courseDetails *models.CourseDetails = new(models.CourseDetails)
	var isReadable bool = false
	var reachedEnd bool = false
	var curValue string = ""
	for {
		curTokenizer := tkn.Next()
		switch {
		case curTokenizer == html.ErrorToken:
			err := tkn.Err()
			if err == io.EOF {
				reachedEnd = true
				break
			}
			return nil, err
		case curTokenizer == html.StartTagToken:
			curToken := tkn.Token()
			if len(curToken.Attr) > 0 && curToken.Attr[0].Key == "id" {
				curValue = curToken.Attr[0].Val
				isReadable = true
			}
		case curTokenizer == html.TextToken:
			if isReadable {
				curToken := tkn.Token()
				data := strings.TrimSpace(curToken.Data)
				switch curValue {
				case "sectionNumber":
					courseDetails.SectionNumber = data
				case "subject":
					courseDetails.Subject = data
				case "courseNumber":
					courseDetails.CourseNumber = data
				case "courseTitle":
					courseDetails.CourseTitle = data
				}
				isReadable = false
			}
		}
		if reachedEnd {
			break
		}
	}
	return courseDetails, nil
}
