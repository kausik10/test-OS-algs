package main

import (
	"fmt"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"math/rand"
	"sync"
	"time"
)

// Customer struct to hold customer details
type Customer struct {
	ID          int
	ArrivalTime time.Time
	ServiceTime time.Duration
	StartTime   time.Time
	EndTime     time.Time
}

// Teller struct to represent a teller
type Teller struct {
	ID           int
	CustomerChan chan Customer
}

// NewTeller creates a new teller
func NewTeller(id int) *Teller {
	return &Teller{
		ID:           id,
		CustomerChan: make(chan Customer),
	}
}

// Work method for teller to process customers
func (t *Teller) Work(wg *sync.WaitGroup, mu *sync.Mutex, customers *[]Customer) {
	defer wg.Done()
	for customer := range t.CustomerChan {
		mu.Lock()
		customer.StartTime = time.Now()
		mu.Unlock()
		fmt.Printf("Customer %d is in Teller %d\n", customer.ID, t.ID)
		time.Sleep(customer.ServiceTime)
		mu.Lock()
		customer.EndTime = time.Now()
		*customers = append(*customers, customer)
		mu.Unlock()
		fmt.Printf("Customer %d leaves Teller %d\n", customer.ID, t.ID)
	}
}

// CalculateMetrics calculates and returns average turnaround time, waiting time, and response time
func CalculateMetrics(customers []Customer) (float64, float64, float64) {
	var totalTurnaroundTime, totalWaitingTime, totalResponseTime time.Duration
	for _, customer := range customers {
		turnaroundTime := customer.EndTime.Sub(customer.ArrivalTime)
		waitingTime := customer.StartTime.Sub(customer.ArrivalTime)
		responseTime := customer.StartTime.Sub(customer.ArrivalTime)

		totalTurnaroundTime += turnaroundTime
		totalWaitingTime += waitingTime
		totalResponseTime += responseTime
	}

	numCustomers := len(customers)
	avgTurnaroundTime := totalTurnaroundTime.Seconds() / float64(numCustomers)
	avgWaitingTime := totalWaitingTime.Seconds() / float64(numCustomers)
	avgResponseTime := totalResponseTime.Seconds() / float64(numCustomers)

	return avgTurnaroundTime, avgWaitingTime, avgResponseTime
}

// SimulateFCFS function to simulate the banking system with FCFS scheduling
func SimulateFCFS(numCustomers int) ([]Customer, float64, float64, float64) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	customerQueue := make(chan Customer, 5) // Limited queue size to simulate full queue scenario
	customers := []Customer{}

	// Create tellers
	tellers := []*Teller{
		NewTeller(1),
		NewTeller(2),
		NewTeller(3),
	}

	// Start tellers
	for _, teller := range tellers {
		wg.Add(1)
		go teller.Work(&wg, &mu, &customers)
	}

	// Function to dispatch customers from the queue to tellers
	go func() {
		for customer := range customerQueue {
			assigned := false
			for !assigned {
				for _, teller := range tellers {
					select {
					case teller.CustomerChan <- customer:
						assigned = true
						break
					default:
					}
					if assigned {
						break
					}
				}
				if !assigned {
					time.Sleep(100 * time.Millisecond) // Wait before retrying to assign
				}
			}
		}

		// Close teller channels after all customers have been dispatched
		for _, teller := range tellers {
			close(teller.CustomerChan)
		}
	}()

	// Generate customers and send to queue
	for i := 1; i <= numCustomers; i++ {
		customer := Customer{
			ID:          i,
			ArrivalTime: time.Now(),
			ServiceTime: time.Duration(rand.Intn(6)+5) * time.Second, // Service time between 5 and 10 seconds
		}
		for {
			select {
			case customerQueue <- customer:
				fmt.Printf("Customer %d enters the Queue\n", customer.ID)
				break
			default:
				fmt.Printf("Queue is FULL. Customer %d is waiting to enter the queue\n", customer.ID)
				time.Sleep(500 * time.Millisecond) // Wait before trying again
			}
			if len(customerQueue) < cap(customerQueue) {
				break
			}
		}
		time.Sleep(time.Duration(rand.Intn(1500)+500) * time.Millisecond) // Random arrival time between 0.5 and 1.5 seconds
	}

	// Close customer queue and wait for all tellers to finish
	close(customerQueue)
	wg.Wait()

	avgTurnaroundTime, avgWaitingTime, avgResponseTime := CalculateMetrics(customers)

	return customers, avgTurnaroundTime, avgWaitingTime, avgResponseTime
}

func main() {
	rand.Seed(time.Now().UnixNano())
	numCustomers := 10
	_, avgTurnaroundTime, avgWaitingTime, avgResponseTime := SimulateFCFS(numCustomers)

	// Print the average metrics
	fmt.Printf("Average Turnaround Time: %.2f seconds\n", avgTurnaroundTime)
	fmt.Printf("Average Waiting Time: %.2f seconds\n", avgWaitingTime)
	fmt.Printf("Average Response Time: %.2f seconds\n", avgResponseTime)

	p := plot.New()

	plotter.DefaultLineStyle.Width = vg.Points(1)
	plotter.DefaultGlyphStyle.Radius = vg.Points(3)

	// Create a plotter.Values for the data
	values := plotter.Values{avgTurnaroundTime, avgWaitingTime, avgResponseTime}

	// Create a bar chart
	barChart, err := plotter.NewBarChart(values, vg.Points(50))
	if err != nil {
		panic(err)
	}

	// Set value labels on bars
	// _labels := []string{
	// 	fmt.Sprintf("Turnaround Time: %.2f", avgTurnaroundTime),
	// 	fmt.Sprintf("Waiting Time: %.2f", avgWaitingTime),
	// 	fmt.Sprintf("Response Time: %.2f", avgResponseTime),
	// }
	// barChart. = labels

	// Add the bar chart to the plot
	p.Add(barChart)

	// Set the X and Y labels
	p.Y.Label.Text = "Time (seconds)"
	p.X.Label.Text = "Metrics"

	// Save the plot to a file
	if err := p.Save(4*vg.Inch, 4*vg.Inch, "metrics.png"); err != nil {
		panic(err)
	}
}
