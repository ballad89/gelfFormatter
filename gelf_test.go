package gelfFormatter

import (
	"testing"

	"errors"

	log "github.com/sirupsen/logrus"
)

func TestWrite(t *testing.T) {

}

func TestWithSingleMessage(t *testing.T) {
	setup()
	log.Info("Short message only.")
}

func TestWithFullMessage(t *testing.T) {
	setup()
	log.Info("Starting application.\n Waiting for request ...")
}

func TestWithAdditionalFields(t *testing.T) {
	setup()

	log.WithFields(log.Fields{
		"animal": "walrus",
	}).Info("animal")
}

func TestWithErrorAdditionalField(t *testing.T) {
	setup()

	log.WithFields(log.Fields{
		"err": errors.New("failed"),
	}).Error("gimmick")
}

func setup() {
	f, err := NewGelfFormatter("application-name")

	if err != nil {
		panic(err)
	}
	log.SetFormatter(f)
}
