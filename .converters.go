package eloverblik

import (
	"fmt"
	"strconv"
	"time"
)

type SimpleTimeSeriesPoint struct {
	Date        time.Time
	Measurement float32
	Quality     string
}

func SimplifiyTimeseries(timeseries TimeSeries) ([]SimpleTimeSeriesPoint, error) {

	// Allocate memory
	datums := make([]SimpleTimeSeriesPoint, 0)
	for _, ts := range timeseries.MyEnergyDataMarketDocument.TimeSeries {
		for _, period := range ts.Periods {
			for _, point := range period.Points {

				duration, err := time.ParseDuration(fmt.Sprintf("%sh", point.Position))
				if err != nil {
					return nil, err
				}

				measurement, err := strconv.ParseFloat(point.Out_Quantity_quantity, 32)
				if err != nil {
					return nil, err
				}

				datums = append(datums, SimpleTimeSeriesPoint{
					Date:        period.TimeInterval.Start.Add(duration + time.Hour),
					Measurement: float32(measurement),
					Quality:     point.Out_Quantity_quality,
				})
			}
		}
	}
	return datums, nil
}
