package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"time"
)

type BarWidget struct {
	Name      string `json:"name"`
	Instance  string `json:"instance,omitempty"`
	FullText  string `json:"full_text"`
	ShortText string `json:"short_text,omitempty"`
	Icon      string `json:"icon,omitempty"`
	Color     string `json:"color,omitempty"`
	Status    string `json:"_status,omitempty"`
}

func batterywidget(widget chan<- *BarWidget, device string) {
	// FIXME: figure out how to not repeat code here

	capacity, err := ioutil.ReadFile(fmt.Sprintf("/sys/class/power_supply/%s/capacity", device))
	if err != nil {
		log.Fatal("Unable to read battery capacity:", err)
	}

	// FIXME: see if there is a better way to do this
	percent := 100
	fmt.Sscanf(string(capacity), "%d", &percent)
	if percent > 100 {
		percent = 100
	}

	status := "normal"
	if percent <= 15 {
		status = "warn"
	}

	widget <- &BarWidget{
		Name:     "battery",
		Instance: device,
		FullText: fmt.Sprintf("%d%%", percent),
		Status:   status,
	}

	c := time.Tick(15 * time.Second)
	for range c {
		capacity, err := ioutil.ReadFile(fmt.Sprintf("/sys/class/power_supply/%s/capacity", device))
		if err != nil {
			log.Fatal("Unable to read battery capacity:", err)
		}

		oldpercent := percent

		// FIXME: see if there is a better way to do this
		percent := 100
		fmt.Sscanf(string(capacity), "%d", &percent)
		if percent > 100 {
			percent = 100
		}

		if percent != oldpercent {
			status := "normal"
			if percent <= 15 {
				status = "warn"
			}

			widget <- &BarWidget{
				Name:     "battery",
				Instance: device,
				FullText: fmt.Sprintf("%d%%", percent),
				Status:   status,
			}
		}
	}
}

func clockwidget(widget chan<- *BarWidget) {
	// FIXME: figure out how to not repeat code here
	now := time.Now()
	widget <- &BarWidget{
		Name:      "clock",
		FullText:  now.Format("Mon Jan 2 15:04"),
		ShortText: now.Format("15:04"),
	}

	// sleep until the next minute
	duration := now.Add(1 * time.Minute).Truncate(time.Minute).Sub(now)
	time.Sleep(duration)

	// FIXME: figure out how to not repeat code here
	now = time.Now()
	widget <- &BarWidget{
		Name:      "clock",
		FullText:  now.Format("Mon Jan 2 15:04"),
		ShortText: now.Format("15:04"),
	}

	c := time.Tick(1 * time.Minute)
	for now := range c {
		widget <- &BarWidget{
			Name:      "clock",
			FullText:  now.Format("Mon Jan 2 15:04"),
			ShortText: now.Format("15:04"),
		}
	}
}

func main() {
	fmt.Print(`{"version":1}[[]`)

	batw := make(chan *BarWidget)
	go batterywidget(batw, "BAT0")

	clockw := make(chan *BarWidget)
	go clockwidget(clockw)

	battery := <-batw
	clock := <-clockw
	widgets := []*BarWidget{battery, clock}

	for {
		// TODO: handle icon and _status

		b, err := json.Marshal(widgets)
		if err != nil {
			log.Fatal("Unable to marshal JSON:", err)
		}

		fmt.Print("," + string(b))

		select {
		case msg := <-batw:
			battery = msg
			widgets = []*BarWidget{battery, clock}
		case msg := <-clockw:
			clock = msg
			widgets = []*BarWidget{battery, clock}
		}
	}

	fmt.Println("]")
}

// vim:set ts=8 sw=8 noet:
