package main

import (
	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/service"

	"github.com/tarm/serial"

	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

type kakuCmd struct {
	state   bool
	group   bool
	channel int
	hwid    int
}

type KikaAccessory struct {
	*accessory.Accessory
	switches []*service.Switch
}

var (
	acc       *KikaAccessory
	cmdChan   chan kakuCmd
	serialDev string
)

func sendKakuPulses(cmd kakuCmd) {
	s, err := serial.OpenPort(&serial.Config{Name: serialDev, Baud: 115200})
	if err != nil {
		log.Fatalf("Could not open serial port: %v", err)
	}

	r := bufio.NewReader(s)
	w := bufio.NewWriter(s)

	if err := w.WriteByte('\n'); err != nil {
		log.Fatalf("Coud not write to serial (1): %v", err)
	}
	if err := w.Flush(); err != nil {
		log.Fatalf("Could not flush: %v", err)
	}

	for {
		line, err := r.ReadBytes(byte('\n'))
		if err != nil {
			log.Fatalf("Could not read from serial: %v", err)
		}
		line = bytes.TrimSpace(line)
		if bytes.Equal(line, []byte("C")) {
			continue
		}
		if bytes.Equal(line, []byte("?")) {
			break
		}
		log.Fatalf("Unexpected line: %s", line)
	}

	if err := w.WriteByte('R'); err != nil {
		log.Fatalf("Could not write to serial (2): %v", err)
	}

	pulses, T := generateKakuPulses(cmd)

	pulses = append(pulses, pulses...)

	if _, err := w.WriteString(fmt.Sprintf("%d\n", len(pulses)+1)); err != nil {
		log.Fatalf("Could not write to serial (3): %v", err)
	}

	for i := 0; i < len(pulses)+1; i++ {
		if _, err := w.WriteString(fmt.Sprintf("%d\n", T)); err != nil {
			log.Fatalf("Could not write to serial (4): %v", err)
		}
		if i == len(pulses) {
			break
		}
		if _, err := w.WriteString(fmt.Sprintf("%d\n", pulses[i])); err != nil {
			log.Fatalf("Could not write to serial (5): %v", err)
		}
	}

	if err := w.Flush(); err != nil {
		log.Fatal("Could not flush (2): %v", err)
	}

	line, err := r.ReadBytes(byte('\n'))
	if err != nil {
		log.Fatalf("Could not read from serial (2): %v", err)
	}
	line = bytes.TrimSpace(line)
	if !bytes.Equal(line, []byte("!")) {
		log.Fatalf("Unexpected line: %s", line)
	}
}

func generateKakuPulses(cmd kakuCmd) ([]int, int) {
	var ret []int = []int{}
	b2i := map[bool]int{true: 1, false: 0}
	T := 265
	bit := [][]int{[]int{T, 5 * T}, []int{5 * T, T}}
	ret = append(ret, T*11)
	for i := 25; i >= 0; i-- {
		ret = append(ret, bit[(cmd.hwid>>uint(i))&1]...)
	}
	ret = append(ret, bit[b2i[cmd.group]]...)
	ret = append(ret, bit[b2i[cmd.state]]...)
	for i := 3; i >= 0; i-- {
		ret = append(ret, bit[(cmd.channel>>uint(i))&1]...)
	}
	ret = append(ret, T*32)
	return ret, T
}

func main() {
	var pin string
	var port int
	var rawIds string
	var storagePath string

	cmdChan = make(chan kakuCmd, 2)

	flag.StringVar(&pin, "pin", "00102003", "pincode")
	flag.IntVar(&port, "port", 0, "Local port to use")
	flag.StringVar(&serialDev, "serial", "/dev/ttyUSB0",
		"path to serial device connected to arduino")
	flag.StringVar(&rawIds, "hwid", "12312312", "hwid(s) of klikaanklikuit group comma separated")
	flag.StringVar(&storagePath, "db", "./db", "path to local storage")

	flag.Parse()

	ids := []int{}
	for _, rawId := range strings.Split(rawIds, ",") {
		id, err := strconv.Atoi(rawId)
		if err != nil {
			log.Fatalf("Parsing hwids: %v", err)
		}
		ids = append(ids, id)
	}

	info := accessory.Info{
		Name: "KlikAanKlikUit",
	}

	acc = new(KikaAccessory)
	acc.Accessory = accessory.New(info, accessory.TypeOther)
	acc.switches = []*service.Switch{}

	for _, id := range ids {
		for i := 0; i < 3; i++ {
			sw := service.NewSwitch()
			acc.switches = append(acc.switches, sw)
			acc.AddService(sw.Service)

			id := id
			i := i
			sw.On.OnValueRemoteUpdate(func(state bool) {
				cmdChan <- kakuCmd{
					hwid:    id,
					channel: i,
					group:   false,
					state:   state,
				}
			})
		}
	}

	var portString = ""
	if port != 0 {
		portString = strconv.Itoa(port)
	}
	config := hc.Config{
		Pin:         pin,
		Port:        portString,
		StoragePath: storagePath,
	}
	t, err := hc.NewIPTransport(config, acc.Accessory)
	if err != nil {
		log.Fatalf("Could not create transport: %v", err)
	}

	go func() {
		for cmd := range cmdChan {
			sendKakuPulses(cmd)
		}
	}()

	hc.OnTermination(func() {
		close(cmdChan)
		t.Stop()
		os.Exit(0)
	})

	t.Start()
}
