package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"time"
	"strconv"
	"strings"
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

func batteryUpdate(device string) (*BarWidget, int) {
	capacity, err := ioutil.ReadFile(fmt.Sprintf("/sys/class/power_supply/%s/capacity", device))
	if err != nil {
		log.Fatal("Unable to read battery capacity:", err)
	}

	percent, _ := strconv.Atoi(strings.TrimSpace(string(capacity)))
	if percent > 100 {
		percent = 100
	}

	status := "normal"
	if percent <= 15 {
		status = "warn"
	}

	return &BarWidget{
		Name:     "battery",
		Instance: device,
		FullText: fmt.Sprintf("%d%%", percent),
		Status:   status,
	}, percent
}

func batteryWidget(widget chan<- *BarWidget, device string) {
	w, percent := batteryUpdate(device)
	widget <- w

	c := time.Tick(15 * time.Second)
	for range c {
		oldpercent := percent
		w, percent := batteryUpdate(device)

		if percent != oldpercent {
			widget <- w
		}
	}
}

func clockUpdate(now time.Time) *BarWidget {
	return &BarWidget{
		Name:      "clock",
		FullText:  now.Format("Mon Jan 2 15:04"),
		ShortText: now.Format("15:04"),
	}
}

func clockWidget(widget chan<- *BarWidget) {
	now := time.Now()
	widget <- clockUpdate(now)

	// sleep until the next minute
	duration := now.Add(1 * time.Minute).Truncate(time.Minute).Sub(now)
	time.Sleep(duration)

	widget <- clockUpdate(time.Now())

	c := time.Tick(1 * time.Minute)
	for now := range c {
		widget <- clockUpdate(now)
	}
}

// TODO: make this generic
func themeBattery(battery *BarWidget) {
	battery.FullText = fmt.Sprintf("\uf3cf %s", battery.FullText)
	if battery.Status == "warn" {
		battery.Color = "#e37170"
	} else {
		battery.Color = "#7f9f7f"
	}
}

// TODO: make this generic
func themeClock(clock *BarWidget) {
	clock.FullText = fmt.Sprintf("\uf017 %s", clock.FullText)
}

func main() {
	fmt.Print(`{"version":1}[[]`)

	batw := make(chan *BarWidget)
	go batteryWidget(batw, "BAT0")

	clockw := make(chan *BarWidget)
	go clockWidget(clockw)

	battery := <-batw
	themeBattery(battery)
	clock := <-clockw
	themeClock(clock)
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
			themeBattery(battery)
			widgets = []*BarWidget{battery, clock}
		case msg := <-clockw:
			clock = msg
			themeClock(clock)
			widgets = []*BarWidget{battery, clock}
		}
	}

	fmt.Println("]")
}

// vim:set ts=8 sw=8 noet:
