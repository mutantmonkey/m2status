package main

import (
	"fmt"
	"github.com/fhs/gompd/mpd"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var handlers map[string]func(*Widget)

func init() {
	handlers = map[string]func(*Widget){
		"mpd":     mpdWidget,
		"wifi":    wifiWidget,
		"battery": batteryWidget,
		"clock":   clockWidget,
	}
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

	status := "normal"
	percent, _ := strconv.Atoi(strings.TrimSpace(string(capacity)))
	if percent > 100 {
		percent = 100
	} else if percent <= 15 {
		status = "warn"
	}

	return &BarWidget{
		Name:     "battery",
		Instance: device,
		FullText: fmt.Sprintf("%d%%", percent),
		Status:   status,
	}, percent
}

func batteryWidget(widget *Widget) {
	device := widget.Config.Args[0]
	file, err := os.Open(fmt.Sprintf("/sys/class/power_supply/%s/capacity", device))
	if err != nil {
		log.Fatal(err)
	}

	w, percent := batteryUpdate(device, file)
	widget.Channel <- w

	c := time.Tick(15 * time.Second)
	for range c {
		oldpercent := percent
		w, percent := batteryUpdate(device, file)

		if percent != oldpercent {
			widget.Channel <- w
		}
	}

	file.Close()
}

func clockUpdate(now time.Time) *BarWidget {
	return &BarWidget{
		Name:      "clock",
		FullText:  now.Format("Mon 2 Jan 15:04"),
		ShortText: now.Format("15:04"),
	}
}

func clockWidget(widget *Widget) {
	now := time.Now()
	widget.Channel <- clockUpdate(now)

	// sleep until the next minute
	duration := now.Add(1 * time.Minute).Truncate(time.Minute).Sub(now)
	time.Sleep(duration)

	widget.Channel <- clockUpdate(time.Now())

	c := time.Tick(1 * time.Minute)
	for now := range c {
		widget.Channel <- clockUpdate(now)
	}
}

func mpdUpdate(addr string) *BarWidget {
	conn, err := mpd.Dial("tcp", addr)
	if err != nil {
		log.Print("MPD update error: ", err)

		return &BarWidget{
			Name:     "mpd",
			Instance: addr,
			FullText: "error",
			Status:   "error",
		}
	}
	defer conn.Close()

	song, err := conn.CurrentSong()
	if err != nil {
		return &BarWidget{
			Name:     "mpd",
			Instance: addr,
			FullText: "error",
		}
	}

	artist, artist_ok := song["Artist"]
	title, title_ok := song["Title"]
	name, name_ok := song["Name"]

	var text string
	if artist_ok && title_ok {
		text = fmt.Sprintf("%s - %s", artist, title)
	} else if title_ok {
		text = title
	} else if name_ok {
		text = name
	} else {
		text = path.Base(song["File"])
	}

	return &BarWidget{
		Name:     "mpd",
		Instance: addr,
		FullText: text,
	}
}

func mpdWidget(widget *Widget) {
	addr := widget.Config.Args[0]

	// push out an empty update to avoid start delays
	widget.Channel <- &BarWidget{
		Name:     "mpd",
		Instance: addr,
	}

	widget.Channel <- mpdUpdate(addr)

	w, err := mpd.NewWatcher("tcp", addr, "", "player")
	if err != nil {
		log.Print("Error creating MPD watcher:", err)
		return
	}
	defer w.Close()

	go func() {
		for err := range w.Error {
			log.Print("MPD watcher error: ", err)
		}
	}()

	for range w.Event {
		widget.Channel <- mpdUpdate(addr)
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

func wifiWidget(widget *Widget) {
	iface := widget.Config.Args[0]
	sock, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		log.Fatal("Unable to get socket:", err)
	}

	w, _ := wifiUpdate(iface, sock)
	widget.Channel <- w

	c := time.Tick(15 * time.Second)
	for range c {
		w, _ := wifiUpdate(iface, sock)
		widget.Channel <- w
	}
}
