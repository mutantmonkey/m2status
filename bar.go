package main

import (
	"fmt"
	"log"
	"encoding/json"
)

type BarWidget struct {
	Name string `json:"name"`
	Instance string `json:"instance,omitempty"`
	FullText string `json:"full_text"`
	ShortText string `json:"short_text,omitempty"`
	Icon string `json:"icon,omitempty"`
	Color string `json:"color,omitempty"`
}

func clock() (*BarWidget) {
	w := &BarWidget{
		Name: "clock",
		FullText: "Thu Sep 24 10:56",
		ShortText: "10:56",
		Icon: "U+f017",
	}
	return w
}

func main() {
	fmt.Print(`{"version":1}[[]`)

	m := clock()
	widgets := []*BarWidget{m}

	b, err := json.Marshal(widgets)
	if err != nil {
		log.Fatal("Unable to marshal JSON:", err)
	}

	fmt.Print("," + string(b))
	fmt.Println("]")
}

// vim:set ts=8 sw=8 noet:
