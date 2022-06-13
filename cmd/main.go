package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/slimcdk/go-eloverblik"
)

func prettyPrint(emp ...interface{}) {
	empJSON, err := json.MarshalIndent(emp, "", "  ")
	if err != nil {
		log.Fatalf(err.Error())
	}
	fmt.Println(string(empJSON))
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	token := os.Getenv("ELO_TOKEN")

	// eloverblik.SetMode(eloverblik.ReleaseMode)
	eloverblik.SetMode(eloverblik.ReleaseMode)

	e, err := eloverblik.CustomerClient(token)
	if err != nil {
		log.Fatalln(err)
	}

	meters, err := e.GetMeteringPoints(false)
	if err != nil {
		log.Fatalln(err)
	}

	var meter eloverblik.MeteringPoints
	for _, m := range meters {
		if strings.Compare(m.MeteringPointID, "571313154390571520") == 0 {
			meter = m
			break
		}
	}
	prettyPrint(meter)

	ts, err := e.GetTimeSeries(
		[]string{meter.MeteringPointID},
		meter.ConsumerStartDate,
		//meter.ConsumerStartDate.Add(eloverblik.MaximumRequestDuration),
		time.Now(),
		eloverblik.Quarter,
	)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("Fetched timeseries")
	prettyPrint(ts[0].Flatten())
	fmt.Println()
}
