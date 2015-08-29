package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
)

const (
	GOOGLE_API = "https://maps.googleapis.com/maps/api/geocode/json"

	EARTH_RADIUS = 6371 // km
)

type (
	// Google Request Object
	GoogleGeocodingAPI struct {
		APIKey string
	}

	// Location Object
	Degree float64

	Location struct {
		Lat     Degree `json:"lat"`
		Lng     Degree `json:"lng"`
		Address string `json:"address"`
	}

	// Google API Response Structs
	GeoLocation struct {
		Lat Degree `json:"lat"`
		Lng Degree `json:"lng"`
	}

	GeoLocationViewPort struct {
		Northeast GeoLocation `json:"northeast"`
		Southwest GeoLocation `json:"southwest"`
	}

	Geometry struct {
		Location     GeoLocation         `json:"location"`
		LocationType string              `json:"location_type"`
		Viewport     GeoLocationViewPort `json:"viewport"`
		Bounds       GeoLocationViewPort `json:"bounds"`
	}

	AddressComponent struct {
		Longname  string   `json:"long_name"`
		Shortname string   `json:"short_name"`
		Types     []string `json:"types"`
	}

	GoogleGeoResult struct {
		AddressComponents []AddressComponent `json:"address_components"`
		FormattedAddress  string             `json:"formatted_address"`
		Geometry          Geometry           `json:"geometry`
		Types             []string           `json:"types"`
	}

	GoogleGeoAPIResponse struct {
		Status  string            `json:"status"`
		Results []GoogleGeoResult `json:"results"`
	}
)

// Degree Functions
func (d Degree) GetRadians() float64 {
	return float64(d) * math.Pi / 180.0
}

// GoogleeocodingAPI Functions
func (g GoogleGeocodingAPI) GetLocation(q string) (*Location, error) {
	request_url := GOOGLE_API + "?sensor=false&address=" + url.QueryEscape(q)

	if len(g.APIKey) > 0 {
		request_url += "&key=" + g.APIKey
	}

	resp, err := http.Get(request_url)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	r := new(GoogleGeoAPIResponse)
	err = json.NewDecoder(resp.Body).Decode(r)

	if err != nil {
		return nil, err
	}

	if len(r.Results) == 0 {
		return nil, nil
	}

	return &Location{
		Lat:     r.Results[0].Geometry.Location.Lat,
		Lng:     r.Results[0].Geometry.Location.Lng,
		Address: r.Results[0].FormattedAddress,
	}, nil

}

func (g GoogleGeocodingAPI) GetDistance(q1, q2, unit string) (float64, error) {
	ch := make(chan *Location)
	locations := []*Location{}

	queries := []string{q1, q2}
	for _, q := range queries {
		go func(q string) {
			loc, _ := g.GetLocation(q)
			ch <- loc
		}(q)
	}

	for {
		r := <-ch
		locations = append(locations, r)
		if len(locations) == 2 {
			dist := geoDistance(*locations[0], *locations[1])
			if unit == "miles" {
				return KM2Miles(dist), nil
			}
			return dist, nil
		}
	}
	return 0.1, nil
}

func KM2Miles(dist float64) float64 {
	return dist * 0.621371192
}

func geoDistance(loc1, loc2 Location) float64 {
	phi1 := loc1.Lat.GetRadians()
	phi2 := loc2.Lat.GetRadians()

	deltaphi := (loc1.Lat - loc2.Lat).GetRadians()
	deltalambda := (loc1.Lng - loc2.Lng).GetRadians()

	a := math.Sin(deltaphi/2)*math.Sin(deltaphi/2) +
		math.Cos(phi1)*math.Cos(phi2)*
			math.Sin(deltalambda/2)*math.Sin(deltalambda/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return EARTH_RADIUS * c
}

func main() {

	var unit = flag.String("unit", "km", "Unit to display distance. (km or miles)")
	var flHelp = flag.Bool("help", false, "Print Usage")

	flag.Parse()

	flag.Usage = func() {
		fmt.Fprint(os.Stdout, "Usage: go-city-distance [OPTIONS] CITY-A CITY-B\n")
		fmt.Fprint(os.Stdout, "       go-city-distance [ --help ]\n\n")
		fmt.Fprint(os.Stdout, "Find the distance between two cities.\n\nOptions:\n")

		flag.CommandLine.SetOutput(os.Stdout)
		flag.PrintDefaults()
	}

	fmt.Println(*unit)

	if *unit != "km" && *unit != "miles" {
		flag.Usage()
		os.Exit(1)
	}

	if *flHelp {
		flag.Usage()
		return
	}

	if len(os.Args) < 3 {
		flag.Usage()
		os.Exit(1)
	}

	var g GoogleGeocodingAPI

	argv := len(os.Args)
	loc1 := os.Args[argv-2]
	loc2 := os.Args[argv-1]
	distance, err := g.GetDistance(loc1, loc2, *unit)

	if err != nil {
		panic(err)
	}

	fmt.Printf("%f\n", distance)
}
