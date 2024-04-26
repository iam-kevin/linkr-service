package linkr

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Converts duration represented as string into seconds
// Inputs can be: 12s, 34d, 18h, 24m
// number must be positive
func ConvertStringDurationToSeconds(strDuration string) (time.Duration, error) {
	dur := strings.TrimSpace(strDuration)

	if len(dur) <= 1 {
		return 0, fmt.Errorf("insufficient string duration to be useful")
	}

	duration, durlen := dur[0:len(dur)-1], strings.ToLower(string(dur[len(dur)-1]))

	// check if the duration string matches a "number" pattern
	ok, err := regexp.Match(`^(\d+)$`, []byte(duration))

	if !ok {
		return 0, fmt.Errorf("likely an invalid duraiton string: %s", duration)
	}

	if err != nil || !ok {
		return 0, err
	}

	// convert to number
	durValue, err := strconv.Atoi(duration)
	if err != nil {
		return 0, err
	}

	if durValue <= 0 {
		return 0, fmt.Errorf("number must be greater than 0")
	}

	// convert to the coresponding second value
	switch durlen {
	case "s":
		{
			return time.Duration(durValue) * time.Second, nil
		}
	case "h":
		{
			return time.Duration(durValue) * time.Hour, nil
		}
	case "m":
		{
			return time.Duration(durValue) * time.Minute, nil
		}
	case "d":
		{
			return time.Duration(durValue) * time.Hour * 24, nil
		}
	}

	return 0, fmt.Errorf("unsupported duration lenght '%s'", durlen)
}
