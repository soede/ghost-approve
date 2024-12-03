package logs

import (
	"log"
	"os"
)

func SetupLogging() {
	log.SetOutput(os.Stdout)

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}
