package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
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

func batteryUpdate(device string, file *os.File) (*BarWidget, int) {
	_, err := file.Seek(0, 0)
	if err != nil {
		log.Fatal(err)
	}

	capacity := make([]byte, 3)
	_, err = file.Read(capacity)
	if err != nil {
		log.Fatal(err)
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
	file, err := os.Open(fmt.Sprintf("/sys/class/power_supply/%s/capacity", device))
	if err != nil {
		log.Fatal(err)
	}

	w, percent := batteryUpdate(device, file)
	widget <- w

	c := time.Tick(15 * time.Second)
	for range c {
		oldpercent := percent
		w, percent := batteryUpdate(device, file)

		if percent != oldpercent {
			widget <- w
		}
	}

	file.Close()
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

func wifiUpdate(iface string, sock int) (*BarWidget, string) {
	const (
		// these values come from linux/wireless.h (V22)
		IFNAMSIZ          = 16
		SIOCGIWESSID      = 0x8B1B
		IW_ESSID_MAX_SIZE = 32
	)

	// create a buffer for the interface name
	ifaceBuf := [IFNAMSIZ]byte{}
	copy(ifaceBuf[:], iface[:])

	// create an empty buffer for the ESSID (the kernel fills this in)
	essidBuf := [IW_ESSID_MAX_SIZE]byte{}

	type essidReq struct {
		Interface [IFNAMSIZ]byte
		Pointer   *[IW_ESSID_MAX_SIZE]byte
		Length    uint16
		Flags     uint16
	}

	req := &essidReq{
		Interface: ifaceBuf,
		Pointer:   &essidBuf,
		Length:    IW_ESSID_MAX_SIZE,
		Flags:     0,
	}

	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(sock), uintptr(SIOCGIWESSID), uintptr(unsafe.Pointer(req)))
	if err != 0 {
		log.Fatal("Syscall failed:", err)
	}

	essid := string(essidBuf[:req.Length])

	return &BarWidget{
		Name:     "wifi",
		Instance: iface,
		FullText: essid,
	}, essid
}

func wifiWidget(widget chan<- *BarWidget, iface string) {
	sock, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		log.Fatal("Unable to get socket:", err)
	}

	w, _ := wifiUpdate(iface, sock)
	widget <- w

	c := time.Tick(15 * time.Second)
	for range c {
		w, _ := wifiUpdate(iface, sock)
		widget <- w
	}
}

// TODO: make this generic
func themeWifi(wifi *BarWidget) {
	if len(wifi.FullText) > 0 {
		wifi.Color = "#dfaf8f"
		wifi.FullText = fmt.Sprintf("\uf405  %s", wifi.FullText)
	}
}

// TODO: make this generic
func themeBattery(battery *BarWidget) {
	battery.FullText = fmt.Sprintf("\uf3cf  %s", battery.FullText)
	if battery.Status == "warn" {
		battery.Color = "#e37170"
	} else {
		battery.Color = "#7f9f7f"
	}
}

// TODO: make this generic
func themeClock(clock *BarWidget) {
	clock.FullText = fmt.Sprintf("\uf017  %s", clock.FullText)
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
