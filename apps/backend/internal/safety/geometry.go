package safety

import (
	"errors"
	"strconv"
	"strings"
)

func buildLineStringWKT(points []Coordinate) (string, error) {
	if len(points) < 2 {
		return "", errors.New("route must contain at least two points")
	}

	var builder strings.Builder
	builder.WriteString("LINESTRING(")
	for index, point := range points {
		if index > 0 {
			builder.WriteString(",")
		}
		builder.WriteString(strconv.FormatFloat(point.Longitude, 'f', 6, 64))
		builder.WriteString(" ")
		builder.WriteString(strconv.FormatFloat(point.Latitude, 'f', 6, 64))
	}
	builder.WriteString(")")

	return builder.String(), nil
}
