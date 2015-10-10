package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"reflect"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Widgets []struct {
		Handler string
		Args    []string
		Color   []string
		Icon    string
	}
}

type WidgetConfig struct {
	Args  []string
	Color []string
	Icon  string
}

type Widget struct {
	Handler func(*Widget)
	Config  *WidgetConfig
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

func NewWidget(handler func(*Widget), config *WidgetConfig) (w *Widget) {
	w = &Widget{
		Handler: handler,
		Config:  config,
		Channel: make(chan *BarWidget),
	}

	go w.Handler(w)
	return
}

func (w *Widget) applyTheme() {
	t := w.Config

	if len(t.Color) == 3 {
		if w.Data.Status == "error" {
			w.Data.Color = t.Color[2]
		} else if w.Data.Status == "warn" {
			w.Data.Color = t.Color[1]
		} else {
			w.Data.Color = t.Color[0]
		}
	} else if len(t.Color) == 1 {
		w.Data.Color = t.Color[0]
	}

	if t.Icon != "" && w.Data.FullText != "" {
		w.Data.FullText = fmt.Sprintf("%s  %s", t.Icon, w.Data.FullText)
	}
}

func main() {
	config := &Config{}
	var configPath string

	defaultConfigPath, err := xdg.ConfigFile("m2bar/config.yml")
	if err != nil {
		log.Print("Unable to get XDG config file path: ", err)
		defaultConfigPath = ""
	}

	flag.StringVar(&configPath, "config", defaultConfigPath,
		"The path to the config file")
	flag.Parse()

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatal("Unable to read config file: ", err)
	} else {
		yaml.Unmarshal(data, &config)
	}

	widgets := make([]*Widget, len(config.Widgets))
	for i, w := range config.Widgets {
		widgets[i] = NewWidget(handlers[w.Handler], &WidgetConfig{
			Args:  w.Args,
			Color: w.Color,
			Icon:  w.Icon,
		})
	}

	fmt.Print(`{"version":1}[[]`)

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
