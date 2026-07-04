package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

func PrintAsJSON(responsBody []byte) error {
	bodyBytes, err := io.ReadAll(bytes.NewReader(responsBody))
	if err != nil {
		return err
	}

	var pretty bytes.Buffer
	err = json.Indent(&pretty, bodyBytes, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(pretty.String())
	return nil
}
