package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/service"
	"github.com/tarm/serial"
	"log"
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

var acc *KikaAccessory
var cmdChan chan kakuCmd

func sendKakuPulses(r *bufio.Reader, w *bufio.Writer, cmd kakuCmd) {
	line, err := r.ReadBytes(byte('\n'))
	if err != nil {
		log.Fatal(err)
	}
	line = bytes.TrimSpace(line)
	if !bytes.Equal(line, []byte("?")) {
		log.Fatalf("Unexpected line: %s", line)
	}

	if err := w.WriteByte('R'); err != nil {
		log.Fatal(err)
	}

	pulses, T := generateKakuPulses(cmd)

	pulses = append(pulses, pulses...)

	if _, err := w.WriteString(fmt.Sprintf("%d\n", len(pulses)+1)); err != nil {
		log.Fatal(err)
	}

	for i := 0; i < len(pulses)+1; i++ {
		if _, err := w.WriteString(fmt.Sprintf("%d\n", T)); err != nil {
			log.Fatal(err)
		}
		if i == len(pulses) {
			break
		}
		if _, err := w.WriteString(fmt.Sprintf("%d\n", pulses[i])); err != nil {
			log.Fatal(err)
		}
	}

	if err := w.Flush(); err != nil {
		log.Fatal(err)
	}

	line, err = r.ReadBytes(byte('\n'))
	if err != nil {
		log.Fatal(err)
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
	var serialDev string
	var port int
	var hwid int
	var storagePath string

	cmdChan = make(chan kakuCmd, 2)

	flag.StringVar(&pin, "pin", "00102003", "pincode")
	flag.IntVar(&port, "port", 0, "Local port to use")
	flag.StringVar(&serialDev, "serial", "/dev/ttyUSB0",
		"path to serial device connected to arduino")
	flag.IntVar(&hwid, "hwid", 12312312, "hwid of klikaanklikuit group")
	flag.StringVar(&storagePath, "db", "./db", "path to local storage")

	flag.Parse()

	s, err := serial.OpenPort(&serial.Config{Name: serialDev, Baud: 115200})
	if err != nil {
		log.Fatalf("Could not open serial port: %v", err)
	}

	info := accessory.Info{
		Name: fmt.Sprintf("KlikAanKlikUit"),
	}

	acc = new(KikaAccessory)
	acc.Accessory = accessory.New(info, accessory.TypeOther)
	acc.switches = []*service.Switch{}

	for i := 0; i < 3; i++ {
		sw := service.NewSwitch()
		acc.switches = append(acc.switches, sw)
		acc.AddService(sw.Service)

		func(i int) {
			sw.On.OnValueRemoteUpdate(func(state bool) {
				cmdChan <- kakuCmd{
					hwid:    hwid,
					channel: i,
					group:   false,
					state:   state,
				}
			})
		}(i)
	}

	var portString = ""
	if port != 0 {
		portString = string(port)
	}
	config := hc.Config{
		Pin:         pin,
		Port:        portString,
		StoragePath: storagePath,
	}
	t, err := hc.NewIPTransport(config, acc.Accessory)
	if err != nil {
		log.Panic(err)
	}

	go func(s *serial.Port) {
		r := bufio.NewReader(s)
		w := bufio.NewWriter(s)

		for cmd := range cmdChan {
			sendKakuPulses(r, w, cmd)
		}
	}(s)

	hc.OnTermination(func() {
		close(cmdChan)
		t.Stop()
	})

	t.Start()
}
