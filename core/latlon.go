package core

import "fmt"

//this is our geospatial-type.
type LatLon struct {
	Lat, Lon float64
}

func (l *LatLon) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%0.6f,%0.6f", l.Lat, l.Lon)), nil
}

func (l *LatLon) UnmarshalText(b []byte) error {
	_, err := fmt.Sscanf(string(b), "%f,%f", &l.Lat, &l.Lon)
	return err
}

func (l *LatLon) IsZero() bool {
	return l.Lat == 0 && l.Lon == 0
}
