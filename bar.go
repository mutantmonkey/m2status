package main

import (
	"encoding/json"
	"fmt"
	"log"
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

func main() {
	fmt.Print(`{"version":1}[[]`)

	wifiw := make(chan *BarWidget)
	go wifiWidget(wifiw, "wlp3s0")

	batw := make(chan *BarWidget)
	go batteryWidget(batw, "BAT0")

	clockw := make(chan *BarWidget)
	go clockWidget(clockw)

	wifi := <-wifiw
	themeWifi(wifi)
	battery := <-batw
	themeBattery(battery)
	clock := <-clockw
	themeClock(clock)
	widgets := []*BarWidget{wifi, battery, clock}

	for {
		// TODO: handle icon and _status

		b, err := json.Marshal(widgets)
		if err != nil {
			log.Fatal("Unable to marshal JSON:", err)
		}

		fmt.Print("," + string(b))

		select {
		case msg := <-wifiw:
			wifi = msg
			themeWifi(wifi)
			widgets = []*BarWidget{wifi, battery, clock}
		case msg := <-batw:
			battery = msg
			themeBattery(battery)
			widgets = []*BarWidget{wifi, battery, clock}
		case msg := <-clockw:
			clock = msg
			themeClock(clock)
			widgets = []*BarWidget{wifi, battery, clock}
		}
	}

	fmt.Println("]")
}

// vim:set ts=8 sw=8 noet:
