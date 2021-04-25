package utils

import "log"

func LogFatalf(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
