package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/DusanKasan/parsemail"
	"github.com/emersion/go-mbox"
	"github.com/joho/godotenv"
	"github.com/kamva/mgm/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	// Setup mgm default config

	DBName := os.Getenv("MONGO_DB")
	DBURI := os.Getenv("MONGO_URI")

	err = mgm.SetDefaultConfig(nil, DBName, options.Client().ApplyURI(DBURI))
	if err != nil {
		panic(err)
	}
	var latestProb ProblemDescription
	if err := mgm.Coll(&ProblemDescription{}).FindOne(mgm.Ctx(), bson.M{}, options.FindOne().SetSort(bson.M{"number": -1})).Decode(&latestProb); err != nil {
		log.Fatal(err)
	}
	LatestProblemNumber = latestProb.Number
	log.Println("Latest Problem Number", LatestProblemNumber)
}

var LatestProblemNumber int

func main() {
	f, err := os.Open("DCP.mbox")
	if err != nil {
		log.Fatal("Unable to open File named DCP.mbox", err)
	}
	defer f.Close()

	mbox := mbox.NewReader(f)
	numOfProblemsEntered := 0
	numOfProblemsSkipped := 0
	for {
		msg, err := mbox.NextMessage()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal("Some Error Occured", err)
		}
		email, err := parsemail.Parse(msg)
		if err != nil {
			log.Fatal("Unable to parse email", err)
		}

		if problemIsAlreadyPresent(&email) {
			numOfProblemsSkipped++
			continue
		}

		problemDescription := makeProblemDescription(&email)
		log.Println("Uploaded Problem", problemDescription.Number)
		if err := mgm.Coll(&ProblemDescription{}).Create(problemDescription); err != nil {
			log.Fatal("Unable To Create Problem:", problemDescription.Number, err)
		}
		numOfProblemsEntered++
	}

	log.Println("All Problems Uploaded Successfully")
	fmt.Println()
	log.Println("Total Number of Problems Uploaded:", numOfProblemsEntered)
	log.Println("Total Number of Emails Skipped:", numOfProblemsSkipped)
}

func problemIsAlreadyPresent(email *parsemail.Email) bool {
	// Check if the email.Subject contains the problem number
	problemNumber := findRegex(`(\d+)`, email.Subject)
	if problemNumber == "" {
		return true
	}
	problemNumberInt, _ := strconv.Atoi(problemNumber)
	return problemNumberInt <= LatestProblemNumber
}

func makeProblemDescription(email *parsemail.Email) *ProblemDescription {
	var problemDescription ProblemDescription
	problemDescription.Date = email.Date
	problemDescription.HTML = email.HTMLBody
	problemDescription.Text = email.TextBody

	numberRegex := `(\d+)`
	// can be one of Easy, Medium, Hard
	difficultyRegex := `(Easy|Medium|Hard)`

	// company name will be in the format of "asked by <company name>"
	companyRegex := `asked by ([a-zA-Z\s]+)`

	// Extract the problem number
	problemDescription.Number, _ = strconv.Atoi(findRegex(numberRegex, email.Subject))

	// Extract the difficulty level
	problemDescription.Difficulty = findRegex(difficultyRegex, email.Subject)

	// Extract the company name
	problemDescription.Company = findRegex(companyRegex, email.TextBody)

	return &problemDescription
}

type ProblemDescription struct {
	mgm.DefaultModel `bson:",inline" json:"_id,omitempty"`
	Number           int       `json:"number" bson:"number"`
	Difficulty       string    `json:"difficulty" bson:"difficulty"`
	Company          string    `json:"company" bson:"company"`
	Text             string    `json:"text" bson:"text"`
	HTML             string    `json:"html" bson:"html"`
	Date             time.Time `json:"date" bson:"date"`
}

func findRegex(regex, text string) string {
	re := regexp.MustCompile(regex)
	match := re.FindStringSubmatch(text)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}
