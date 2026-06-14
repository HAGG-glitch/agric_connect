package weather

import "strings"

type DistrictCoordinates struct {
	Name      string
	Latitude  float64
	Longitude float64
}

var Districts = map[string]DistrictCoordinates{
	"bo":                  {Name: "Bo", Latitude: 7.9647, Longitude: -11.7383},
	"bombali":             {Name: "Bombali", Latitude: 9.4963, Longitude: -11.9788},
	"bonthe":              {Name: "Bonthe", Latitude: 7.5264, Longitude: -12.5050},
	"falaba":              {Name: "Falaba", Latitude: 9.8614, Longitude: -11.3181},
	"kailahun":            {Name: "Kailahun", Latitude: 8.2767, Longitude: -10.5728},
	"kambia":              {Name: "Kambia", Latitude: 9.1167, Longitude: -12.9167},
	"karene":              {Name: "Karene", Latitude: 9.2989, Longitude: -12.5228},
	"kenema":              {Name: "Kenema", Latitude: 7.8767, Longitude: -11.1928},
	"koinadugu":           {Name: "Koinadugu", Latitude: 9.5500, Longitude: -11.3667},
	"kono":                {Name: "Kono", Latitude: 8.7000, Longitude: -10.9833},
	"moyamba":             {Name: "Moyamba", Latitude: 8.1600, Longitude: -12.4300},
	"port loko":           {Name: "Port Loko", Latitude: 8.7667, Longitude: -12.8667},
	"pujehun":             {Name: "Pujehun", Latitude: 7.3567, Longitude: -11.7183},
	"tonkolili":           {Name: "Tonkolili", Latitude: 8.9167, Longitude: -11.8667},
	"western area urban":  {Name: "Western Area Urban", Latitude: 8.4875, Longitude: -13.2344},
	"western area rural":  {Name: "Western Area Rural", Latitude: 8.4200, Longitude: -13.1600},
}

var SupportedDistricts = []string{
	"Bo", "Bombali", "Bonthe", "Falaba", "Kailahun", "Kambia",
	"Karene", "Kenema", "Koinadugu", "Kono", "Moyamba",
	"Port Loko", "Pujehun", "Tonkolili",
	"Western Area Urban", "Western Area Rural",
}

func GetDistrict(name string) (DistrictCoordinates, bool) {
	d, ok := Districts[strings.ToLower(name)]
	return d, ok
}

func IsValidDistrict(name string) bool {
	_, ok := Districts[strings.ToLower(name)]
	return ok
}
