package firehose

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/noaa/consumer"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/nu7hatch/gouuid"
	"os"
	"strconv"
)

var CfClientObj *cfclient.Client

func OpenFirehose(firehoseSubscriptionId string) (<-chan *events.Envelope, error) {

	var err error

	apiEndpoint := os.Getenv("API_ENDPOINT")
	userName := os.Getenv("USER_ID")
	password := os.Getenv("USER_PASSWORD")
	skipSSLValidation, err := strconv.ParseBool(os.Getenv("SKIP_SSL_VALIDATION"))
	if err != nil {
		skipSSLValidation = false
	}

	if apiEndpoint == "" || userName == "" || password == "" {
		return nil, errors.New("Must set environment variables API_ENDPOINT, USER_ID, USER_PASSWORD")
	}

	fmt.Fprintf(os.Stdout, "Authenticating API:%s Credentials:%s Skip_SSL %t\n", apiEndpoint, userName, skipSSLValidation)

	c := cfclient.Config{
		ApiAddress:        apiEndpoint,
		Username:          userName,
		Password:          password,
		SkipSslValidation: skipSSLValidation,
	}

	CfClientObj, err = cfclient.NewClient(&c)
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(os.Stdout, "Opening Firehose api %s\n", CfClientObj.Endpoint.DopplerEndpoint)

	connection := consumer.New(CfClientObj.Endpoint.DopplerEndpoint, &tls.Config{InsecureSkipVerify: c.SkipSslValidation}, nil)

	connection.SetDebugPrinter(ConsoleDebugPrinter{})

	u, err := uuid.NewV4()
	if err == nil {
		firehoseSubscriptionId += u.String()
	}

	fmt.Fprintf(os.Stdout, "Connecting to firehose with subscriptionID %s\n", firehoseSubscriptionId)

	token, err := CfClientObj.GetToken()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failure getting token: %s\n", err)
		return nil, err
	}

	msgChan, errorChan := connection.Firehose(firehoseSubscriptionId, token)

	// Dont understand this one - why a go routine? - but if not, it hangs - should return err instead of exit
	//
	go func() {
		for err := range errorChan {
			fmt.Fprintf(os.Stderr, "Error starting firehose:%v\n", err.Error())
			os.Exit(-1)
		}
	}()

	fmt.Fprintln(os.Stdout, "Firehose opened")

	return msgChan, nil
}

func AppName(guid string) (string, error) {

	app, err := CfClientObj.AppByGuid(guid)
	return app.Name, err

}

type ConsoleDebugPrinter struct{}

func (c ConsoleDebugPrinter) Print(title, dump string) {
	// fmt.Fprintln(os.Stdout, title)
	// fmt.Fprintln(os.Stdout, dump)
}
