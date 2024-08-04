package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type Driver struct {
	ID    string      `json:"id"`
	Route []*Delivery `json:"-"`
}

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Delivery struct {
	ID          int     `json:"id"`
	Source      Point   `json:"point"`
	Destination Point   `json:"destination"`
	Time        float64 `json:"time"`
	Assigned    *Driver `json:"driver"`
}

type Savings struct {
	SourceID      int     `json:"source_id"`
	DestinationID int     `json:"destination_id"`
	Amount        float64 `json:"savings"`
}

// Type used to sort slices of Savings
type BySaving []Savings

func (s BySaving) Len() int           { return len(s) }
func (a BySaving) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a BySaving) Less(i, j int) bool { return a[i].Amount < a[j].Amount }

var Depot = Point{X: 0.0, Y: 0.0}

const MaxTime = 12.0 * 60.0

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: vrp [file input]")
	}

	fileName := os.Args[1]

	routes, err := parse(fileName)
	if err != nil {
		log.Fatal(err)
	}

	drivers := solve(routes)
	printSolution(drivers)
}

func solve(routes map[int]*Delivery) []*Driver {
	s := savings(routes)

	drivers := make([]*Driver, 0)

	for _, link := range s {
		load1 := routes[link.SourceID]
		load2 := routes[link.DestinationID]

		switch {

		// Neither load is assigned
		case load1.Assigned == nil && load2.Assigned == nil:
			arr := make([]*Delivery, 2)
			arr[0] = load1
			arr[1] = load2
			cost := computeTime(arr)

			if cost <= MaxTime {
				driver := Driver{
					ID:    uuid.Must(uuid.NewV7()).String(),
					Route: make([]*Delivery, 0),
				}
				driver.Route = append(driver.Route, load1)
				driver.Route = append(driver.Route, load2)
				drivers = append(drivers, &driver)
				load1.Assigned = &driver
				load2.Assigned = &driver

			}

		// Load 1 is assigned, but load 2 is not
		case load1.Assigned != nil && load2.Assigned == nil:
			driver := load1.Assigned
			i := indexOf(load1, driver.Route)

			// if node is the last node of route
			if i == len(driver.Route)-1 {
				// check constraints
				arr := make([]*Delivery, 0)
				arr = append(arr, driver.Route...)
				arr = append(arr, load2)
				cost := computeTime(arr)
				if cost <= MaxTime {
					driver.Route = append(driver.Route, load2)
					load2.Assigned = driver
				}
			}

		// Load 2 is assigned, but load 1 is not
		case load1.Assigned == nil && load2.Assigned != nil:
			driver := load2.Assigned
			i := indexOf(load2, driver.Route)
			// if node is the first node of route
			if i == 0 {
				// check constraints
				arr := make([]*Delivery, 0)
				arr = append(arr, driver.Route...)
				arr = append(arr, load1)
				cost := computeTime(arr)
				if cost <= MaxTime {
					driver.Route = append(driver.Route, load1)
					load1.Assigned = driver
				}
			}

		// Both loads are already assigned
		default:
			driver1 := load1.Assigned
			i1 := indexOf(load1, driver1.Route)

			driver2 := load2.Assigned
			i2 := indexOf(load2, driver2.Route)

			// if node1 is the last node of its route and node 2 is the first node of its route and the routes are different
			if (i1 == len(driver1.Route)-1) && (i2 == 0) && (driver1.ID != driver2.ID) {
				arr := make([]*Delivery, 0)
				arr = append(arr, driver1.Route...)
				arr = append(arr, driver2.Route...)
				cost := computeTime(arr)
				if cost <= MaxTime {
					driver1.Route = append(driver1.Route, driver2.Route...)
					for _, load := range driver2.Route {
						load.Assigned = driver1
					}
					drivers = removeDriver(drivers, *driver2)
				}
			}
		}
	}

	// Assign all unassigned routes to individual drivers
	for _, load := range routes {
		if load.Assigned == nil {
			driver := Driver{
				ID:    uuid.Must(uuid.NewV7()).String(),
				Route: make([]*Delivery, 0),
			}
			driver.Route = append(driver.Route, load)
			drivers = append(drivers, &driver)
			load.Assigned = &driver
		}
	}

	return drivers
}

// Print out the solution in the required format
func printSolution(drivers []*Driver) {
	for _, d := range drivers {
		ids := make([]string, len(d.Route))
		for i, r := range d.Route {
			ids[i] = fmt.Sprintf("%d", r.ID)
		}
		fmt.Printf("[%s]\n", strings.Join(ids, ","))
	}
}

// Remove a driver from a slice of drivers
func removeDriver(drivers []*Driver, d Driver) []*Driver {
	idx := -1

	for i, dr := range drivers {
		if dr.ID == d.ID {
			idx = i
			break
		}
	}

	if idx == -1 {
		return nil
	}

	return append(drivers[:idx], drivers[idx+1:]...)
}

// Find the index of a delivery in a slice based on ID
func indexOf(d *Delivery, arr []*Delivery) int {
	for i, n := range arr {
		if n.ID == d.ID {
			return i
		}
	}
	return -1
}

// Compute the total time of a set of deliveries
func computeTime(nodes []*Delivery) float64 {
	if len(nodes) == 0 {
		return 0.0
	}

	time := 0.0
	for i := 0; i < len(nodes); i++ {
		time += nodes[i].Time
		if i != (len(nodes) - 1) {
			time += distance(nodes[i].Destination, nodes[i+1].Source)
		}
	}

	time += distance(Depot, nodes[0].Source)
	time += distance(nodes[len(nodes)-1].Destination, Depot)

	return time
}

// Compute the Clark-Wright savings
// https://web.mit.edu/urban_or_book/www/book/chapter6/6.4.12.html
func savings(routes map[int]*Delivery) []Savings {
	savings := make([]Savings, 0)

	for _, i := range routes {
		for _, j := range routes {
			if i == j {
				continue
			}

			// Formula: savings = D(i.dropoff, 0) + D(0, j.pickup) - D(i.dropoff, j.pickup)
			saving := distance(i.Destination, Depot) + distance(Depot, j.Source) - distance(i.Destination, j.Source)
			savings = append(savings, Savings{
				SourceID:      i.ID,
				DestinationID: j.ID,
				Amount:        saving,
			})
		}
	}

	// Sort in descending order
	sort.Slice(savings, func(i, j int) bool {
		return savings[i].Amount > savings[j].Amount
	})

	return savings
}

// Simple euclidean distance betweeen two points in Cartesian space
func distance(i, j Point) float64 {
	return math.Sqrt((i.X-j.X)*(i.X-j.X) + (i.Y-j.Y)*(i.Y-j.Y))
}

// Parse an input file, returning a map of Deliveries indexed by ID
func parse(fName string) (map[int]*Delivery, error) {
	f, err := os.Open(fName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// File is space delimited. Treat as a CSV.
	reader := csv.NewReader(f)
	reader.Comma = ' '

	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) < 2 {
		return nil, errors.New("improperly formatted input file")
	}

	ret := make(map[int]*Delivery, 0)

	// Be sure to skip the first line
	for i := 1; i < len(records); i++ {
		if len(records[i]) != 3 {
			return nil, errors.New("improperly formatted input file")
		}
		id := records[i][0]
		source := records[i][1]
		dest := records[i][2]

		idInt, err := strconv.Atoi(id)
		if err != nil {
			return nil, errors.New("improperly formatted input file")
		}

		// Parse source and destination coordinates and remove the parentheses
		sourceCoord := strings.Split(strings.ReplaceAll(strings.ReplaceAll(source, "(", ""), ")", ""), ",")
		if len(sourceCoord) != 2 {
			return nil, errors.New("improperly formatted input file")
		}
		destCoord := strings.Split(strings.ReplaceAll(strings.ReplaceAll(dest, "(", ""), ")", ""), ",")
		if len(destCoord) != 2 {
			return nil, errors.New("improperly formatted input file")
		}

		// Convert coordinates into floating points (using float64 cuz no real reason not to)
		sourceX, err := strconv.ParseFloat(sourceCoord[0], 64)
		if err != nil {
			return nil, errors.New("improperly formatted input file")
		}
		sourceY, err := strconv.ParseFloat(sourceCoord[1], 64)
		if err != nil {
			return nil, errors.New("improperly formatted input file")
		}

		destX, err := strconv.ParseFloat(destCoord[0], 64)
		if err != nil {
			return nil, errors.New("improperly formatted input file")
		}
		destY, err := strconv.ParseFloat(destCoord[1], 64)
		if err != nil {
			return nil, errors.New("improperly formatted input file")
		}

		d := Delivery{
			ID: idInt,
			Source: Point{
				X: sourceX,
				Y: sourceY,
			},
			Destination: Point{
				X: destX,
				Y: destY,
			},
		}

		// Precompute the drive time
		d.Time = distance(d.Source, d.Destination)

		ret[idInt] = &d
	}

	return ret, nil
}
