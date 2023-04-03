package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"time"

	"github.com/charmbracelet/log"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
)

func main() {
	inputDirFlag := flag.String("input-dir", "", "Directory containing the input JSON files")
	startDateFlag := flag.String("start-date", "", "Start of time range to consider, example: 2020-01-01T00:00:00")
	endDateFlag := flag.String("end-date", "", "End of time range to consider")
	latitudeFlag := flag.String("latitude", "", "Latitude of the location")
	longitudeFlag := flag.String("longitude", "", "Longitude of the location")
	toleranceFlag := flag.String("tolerance", "1000", "Radius around location in meters, contained places are considered as the location ")
	verboseFlag := flag.Bool("verbose", false, "Verbose output")
	printDatesFlag := flag.Bool("print-dates", false, "Print dates")

	flag.Parse()

	logLevel := log.InfoLevel
	if *verboseFlag {
		logLevel = log.DebugLevel
	}

	log.SetDefault(log.NewWithOptions(os.Stderr, log.Options{
		Level: logLevel,
	}))

	// TODO: Validate input

	latitude, err := strconv.ParseFloat(*latitudeFlag, 64)
	if err != nil {
		log.Error("Could not parse latitude", "err", err)
	}

	longitude, err := strconv.ParseFloat(*longitudeFlag, 64)
	if err != nil {
		log.Error("Could not parse longitude", "err", err)

	}

	tolerance, err := strconv.ParseFloat(*toleranceFlag, 64)
	if err != nil {
		log.Error("Could not parse tolerance", "err", err)
	}

	startDate, err := time.ParseInLocation(time.RFC3339, *startDateFlag, time.Local)
	if err != nil {
		log.Error("Could not parse start date", "err", err)
	}

	endDate, err := time.ParseInLocation(time.RFC3339, *endDateFlag, time.Local)
	if err != nil {
		log.Error("Could not parse end date", "err", err)
	}

	fileNames, err := listFilesRecursively(*inputDirFlag)
	if err != nil {
		log.Error("Could not list files", "err", err)
	}

	officeLocation := orb.Point{latitude, longitude}
	daysInTheOffice := make(dayMap)

	for _, fileName := range fileNames {
		processFile(fileName, startDate, endDate, officeLocation, tolerance, daysInTheOffice)
	}

	log.Infof("You have been in the office on %d day(s) of which %d have been working days.", len(daysInTheOffice), daysInTheOffice.CountWorkingDays())

	if *printDatesFlag {
		list := daysInTheOffice.ToSlice()

		sort.Strings(list)

		for _, date := range list {
			fmt.Print(date)

			if !daysInTheOffice[date] {
				fmt.Print(" (weekend)")
			}

			fmt.Print("\n")
		}
	}
}

// dayMap maps a stringified date to a boolean indicating whether it was a working day
type dayMap map[string]bool

func (d dayMap) Add(t time.Time) {
	date := t.Format("2006-01-02")
	d[date] = t.Weekday() != time.Saturday && t.Weekday() != time.Sunday
}

func (d dayMap) ToSlice() []string {
	slice := make([]string, 0, len(d))

	for key := range d {
		slice = append(slice, key)
	}

	return slice
}

func (d dayMap) CountWorkingDays() int {
	count := 0

	for _, isWorkingDay := range d {
		if isWorkingDay {
			count++
		}
	}

	return count
}

func processFile(fileName string, startDate, endDate time.Time, officeLocation orb.Point, tolerance float64, daysInTheOffice dayMap) {
	logger := log.With("file", fileName)

	file, err := os.OpenFile(fileName, os.O_RDONLY, 0)
	if err != nil {
		logger.Error("Could not open file", "err", err)
	}

	places, err := ParseTimelineInput(file)
	if err != nil {
		logger.Error("Could not parse file", "err", err)
	}

	placesProcessed := 0

	for _, place := range places {
		if place.Duration.End.Before(startDate) || place.Duration.Start.After(endDate) {
			// We expect entries to be in sorted order, so we could break here.
			// But as we do not know for sure we instead go the extra mile.
			continue
		}

		loc := orb.Point{place.CenterLatE7 / 1e7, place.CenterLngE7 / 1e7}

		distance := geo.DistanceHaversine(officeLocation, loc)

		if distance <= tolerance {
			daysInTheOffice.Add(place.Duration.Start)
		}

		placesProcessed++
	}

	logger.Debugf("Found %d visits to places in file of which %d have been (partially) within the given time range", len(places), placesProcessed)
}

func listFilesRecursively(inputDir string) ([]string, error) {
	var list []string

	var readDir func(string) error
	readDir = func(inputDir string) error {
		entries, err := os.ReadDir(inputDir)
		if err != nil {
			return fmt.Errorf("could not read directory %s: %w", inputDir, err)
		}

		for _, entry := range entries {
			fullPath := path.Join(inputDir, entry.Name())

			if entry.IsDir() {
				err := readDir(fullPath)
				if err != nil {
					return err
				}
			} else {
				list = append(list, fullPath)
			}
		}

		return nil
	}

	if err := readDir(inputDir); err != nil {
		return nil, err
	}

	return list, nil
}

func ParseTimelineInput(input io.Reader) ([]*timelineVisitedPlace, error) {
	type wrapper struct {
		TimelineObjects []struct {
			PlaceVisit *timelineVisitedPlace `json:"placeVisit"`
		} `json:"timelineObjects"`
	}

	var w wrapper

	if err := json.NewDecoder(input).Decode(&w); err != nil {
		return nil, fmt.Errorf("decoding JSON: %w", err)
	}

	// Remove nil entries, i.e. entries that are not place visits but activity segments or something else
	var result []*timelineVisitedPlace
	for _, entry := range w.TimelineObjects {
		if entry.PlaceVisit != nil {
			result = append(result, entry.PlaceVisit)
		}
	}

	return result, nil
}

type timelineVisitedPlace struct {
	Location struct {
		LatitudeE7  int    `json:"latitudeE7"`
		LongitudeE7 int    `json:"longitudeE7"`
		Address     string `json:"address"`
		Name        string `json:"name"`
	}
	Duration struct {
		Start time.Time `json:"startTimestamp"`
		End   time.Time `json:"endTimestamp"`
	}
	VisitConfidence int     `json:"visitConfidence"`
	CenterLatE7     float64 `json:"centerLatE7"`
	CenterLngE7     float64 `json:"centerLngE7"`
}
