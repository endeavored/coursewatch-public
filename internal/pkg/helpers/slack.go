package helpers

import (
	"encoding/json"
	"fmt"

	"github.com/endeavored/coursewatch/internal/pkg/models"
	"github.com/valyala/fasthttp"
)

const DMEndpoint = "https://slack.com/api/chat.postMessage"

/*
This function will be deprecated soon, do not use
*/
func SendToSlack(webhooks []string, courseDetails *models.CourseDetails, courseAvailability *models.CourseAvailability) {
	for _, webhook := range webhooks {
		go sendToSlackWebhook(webhook, courseDetails, courseAvailability)
	}
}

func SendToSlackDM(channel string, courseDetails *models.CourseDetails, courseAvailability *models.CourseAvailability) {
	go initDirectMessageBlocks([]string{channel}, courseDetails, courseAvailability)
}

func initDirectMessageBlocks(channels []string, courseDetails *models.CourseDetails, courseAvailability *models.CourseAvailability) {
	isWaitlist := courseAvailability.WAvailable > 0 || courseAvailability.WActual < courseAvailability.WCapacity
	fmt.Println(courseAvailability.WAvailable)
	//isWaitlist := false
	msg := fmt.Sprintf("(%s) %s - %s has opened a", courseDetails.SectionNumber, courseDetails.Subject, courseDetails.CourseNumber)
	if isWaitlist {
		msg = msg + " waitlist spot"
	} else {
		msg = msg + "n enrollment spot"
	}
	var blocks []models.SlackBlock = make([]models.SlackBlock, 0)
	blocks = append(blocks, models.SlackBlock{
		Type: "header",
		Text: &models.SlackText{
			Type: "plain_text",
			Text: msg,
		},
	})
	blocks = append(blocks, models.SlackBlock{
		Type: "section",
		Text: &models.SlackText{
			Type: "plain_text",
			Text: fmt.Sprintf("CRN %s", courseDetails.CRN),
		},
	})
	var block *models.SlackBlock = new(models.SlackBlock)
	block.Type = "section"
	var fields []models.SlackText = make([]models.SlackText, 0)
	if !isWaitlist {
		fields = append(fields, models.SlackText{
			Type: "mrkdwn",
			Text: "*Enrollment Actual*\n" + fmt.Sprint(courseAvailability.EActual),
		})
		fields = append(fields, models.SlackText{
			Type: "mrkdwn",
			Text: "*Enrollment Maximum*\n" + fmt.Sprint(courseAvailability.EMax),
		})
		fields = append(fields, models.SlackText{
			Type: "mrkdwn",
			Text: "*Enrollment Seats Available*\n" + fmt.Sprint(courseAvailability.EAvailable),
		})
	} else {
		fields = append(fields, models.SlackText{
			Type: "mrkdwn",
			Text: "*Waitlist Actual*\n" + fmt.Sprint(courseAvailability.WActual),
		})
		fields = append(fields, models.SlackText{
			Type: "mrkdwn",
			Text: "*Waitlist Maximum*\n" + fmt.Sprint(courseAvailability.WCapacity),
		})
		fields = append(fields, models.SlackText{
			Type: "mrkdwn",
			Text: "*Waitlist Spots Available*\n" + fmt.Sprint(courseAvailability.WAvailable),
		})
	}
	block.Fields = &fields
	blocks = append(blocks, *block)
	go sendMassDirectMessage(channels, blocks)
}

func sendMassDirectMessage(channels []string, blocks []models.SlackBlock) {
	for _, channel := range channels {
		go sendDirectMessage(channel, blocks)
	}
}

func sendDirectMessage(channel string, blocks []models.SlackBlock) {
	dmData := &models.SlackDirectMessageData{
		Token:   "",
		Channel: channel,
		Blocks:  blocks,
	}
	postData, err := json.Marshal(dmData)
	if err == nil {
		postRequest(DMEndpoint, postData)
	} else {
		fmt.Printf("ERROR MARSHALING SLACK DIRECT MSG: %v", err)
	}
}

func sendToSlackWebhook(webhook string, courseDetails *models.CourseDetails, courseAvailability *models.CourseAvailability) {
	//isWaitlist := false
	isWaitlist := courseAvailability.WAvailable > 0 && !(courseAvailability.WActual == 0 && courseAvailability.EnrollmentAvailable)
	msg := fmt.Sprintf("(%s) %s - %s has opened a", courseDetails.SectionNumber, courseDetails.Subject, courseDetails.CourseNumber)
	if isWaitlist {
		msg = msg + " waitlist spot"
	} else {
		msg = msg + "n enrollment spot"
	}
	var blocks []models.SlackBlock = make([]models.SlackBlock, 0)
	blocks = append(blocks, models.SlackBlock{
		Type: "header",
		Text: &models.SlackText{
			Type: "plain_text",
			Text: msg,
		},
	})
	blocks = append(blocks, models.SlackBlock{
		Type: "section",
		Text: &models.SlackText{
			Type: "plain_text",
			Text: fmt.Sprintf("CRN %s", courseDetails.CRN),
		},
	})
	var block *models.SlackBlock = new(models.SlackBlock)
	block.Type = "section"
	var fields []models.SlackText = make([]models.SlackText, 0)
	if !isWaitlist {
		fields = append(fields, models.SlackText{
			Type: "mrkdwn",
			Text: "*Enrollment Actual*\n" + fmt.Sprint(courseAvailability.EActual),
		})
		fields = append(fields, models.SlackText{
			Type: "mrkdwn",
			Text: "*Enrollment Maximum*\n" + fmt.Sprint(courseAvailability.EMax),
		})
		fields = append(fields, models.SlackText{
			Type: "mrkdwn",
			Text: "*Enrollment Seats Available*\n" + fmt.Sprint(courseAvailability.EAvailable),
		})
	} else {
		fields = append(fields, models.SlackText{
			Type: "mrkdwn",
			Text: "*Waitlist Actual*\n" + fmt.Sprint(courseAvailability.WActual),
		})
		fields = append(fields, models.SlackText{
			Type: "mrkdwn",
			Text: "*Waitlist Maximum*\n" + fmt.Sprint(courseAvailability.WCapacity),
		})
		fields = append(fields, models.SlackText{
			Type: "mrkdwn",
			Text: "*Waitlist Spots Available*\n" + fmt.Sprint(courseAvailability.WAvailable),
		})
	}
	block.Fields = &fields
	blocks = append(blocks, *block)
	webhookData := &models.SlackWebhookData{
		Text:   msg,
		Blocks: blocks,
	}
	postData, err := json.Marshal(webhookData)
	if err == nil {
		postRequest(webhook, postData)
	} else {
		fmt.Printf("ERROR MARSHALING SLACK MSG: %v", err)
	}
}

func postRequest(uri string, body []byte) {
	req := fasthttp.AcquireRequest()
	req.SetBody(body)
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.SetRequestURI(uri)
	res := fasthttp.AcquireResponse()
	if err := fasthttp.Do(req, res); err != nil {
		fmt.Printf("ERROR SENDING TO SLACK: %v", err)
	}
	fasthttp.ReleaseRequest(req)

	// Do something with body.
	fmt.Println(string(res.Body()))

	fasthttp.ReleaseResponse(res) // Only when you are done with body!

}
