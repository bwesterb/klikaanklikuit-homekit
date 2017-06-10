KlikaanKlikuit-HomeKit bridge
-----------------------------

Connects KlikaanKlikuit RF switches to homekit using a server
connected to an arduino connected to a 433.92Mhz transmitter.
[Demo video](https://www.youtube.com/watch?v=mdhz_0Kp_ns)

How to:

 1. Load arduino.ino on the Arduino Uno.
 2. Connect the 433.92Mhz transmitter data line to the arduino on pin 8
    (I use the cheap fs1000a RF-transmitter.)
 3. Install go on the server.
 4. Run `go get github.com/bwesterb/klikaanklikuit-homekit`
 5. Figure out the hardware ID of the KlikaanKlikuit devices.  This is easy
    with [rtl_433](https://github.com/merbanan/rtl_433) if
    you have (a friend with a) software-defined radio receiver.
 6. Run the bridge: `$GOPATH/bin/klikaanklikuit-homekit -serial /dev/ttyDevOfArduino -pin 00102003 -hwid 123123`
