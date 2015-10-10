package main

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
)

type Config struct {
	Widgets []map[string]string
	Theme   map[string]*WidgetTheme
}

type WidgetTheme struct {
	Color []string
	Icon  string
}

type Widget struct {
	Handler func(*Widget)
	Theme   *WidgetTheme
	Args    []interface{}
	Channel chan *BarWidget
	Data    *BarWidget
}

type BarWidget struct {
	Name      string `json:"name"`
	Instance  string `json:"instance,omitempty"`
	FullText  string `json:"full_text"`
	ShortText string `json:"short_text,omitempty"`
	Color     string `json:"color,omitempty"`
	Status    string `json:"_status,omitempty"`
}

func NewWidget(handler func(*Widget), theme *WidgetTheme, args ...interface{}) (w *Widget) {
	w = &Widget{
		Handler: handler,
		Theme:   theme,
		Args:    args,
		Channel: make(chan *BarWidget),
	}

	go w.Handler(w)
	return
}

func (w *Widget) applyTheme() {
	if len(w.Theme.Color) == 3 {
		if w.Data.Status == "error" {
			w.Data.Color = w.Theme.Color[2]
		} else if w.Data.Status == "warn" {
			w.Data.Color = w.Theme.Color[1]
		} else {
			w.Data.Color = w.Theme.Color[0]
		}
	} else if len(w.Theme.Color) == 1 {
		w.Data.Color = w.Theme.Color[0]
	}

	if w.Theme.Icon != "" && w.Data.FullText != "" {
		w.Data.FullText = fmt.Sprintf("%s  %s", w.Theme.Icon, w.Data.FullText)
	}
}

func main() {
	fmt.Print(`{"version":1}[[]`)

	widgets := []*Widget{
		NewWidget(wifiWidget, &WidgetTheme{
			Color: []string{"#dfaf8f"},
			Icon:  "\uf405",
		}, "wlp3s0"),
		NewWidget(batteryWidget, &WidgetTheme{
			Color: []string{"#7f9f7f", "#e37170", "#e37170"},
			Icon:  "\uf3cf",
		}, "BAT0"),
		NewWidget(clockWidget, &WidgetTheme{
			Icon: "\uf017",
		}, nil),
	}

	output := make([]*BarWidget, len(widgets))
	cases := make([]reflect.SelectCase, len(widgets))
	for i, w := range widgets {
		w.Data = <-w.Channel
		w.applyTheme()
		output[i] = w.Data

		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(w.Channel)}
	}

	for {
		b, err := json.Marshal(output)
		if err != nil {
			log.Fatal("Unable to marshal JSON:", err)
		}

		fmt.Print("," + string(b))

		chosen, _, _ := reflect.Select(cases)
		widgets[chosen].Data = <-widgets[chosen].Channel
		widgets[chosen].applyTheme()
		output[chosen] = widgets[chosen].Data
	}

	fmt.Println("]")
}
