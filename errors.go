package eloverblik

import (
	"fmt"
)

func ErrorClientConnection(status string) error {
	return fmt.Errorf("could't connect to eloverblik: %s", status)
}
