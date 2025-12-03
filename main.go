package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"gitlab.com/gomidi/midi/v2"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
)

func main() {
	const (
		// Blofeld listens on channel 5 (0-based value 4).
		blofeldChannel uint8 = 4
		// Blofeld SysEx device ID often matches the MIDI channel (5 -> 0x04).
		blofeldDeviceID byte = 0x00
		nameHint             = "blofeld"
	)

	log.Println("Available MIDI outputs:")
	log.Print(midi.GetOutPorts().String())

	portIdx, err := findOutPort(nameHint)
	if err != nil {
		log.Fatalf("could not find Blofeld MIDI out port: %v", err)
	}

	inPortIdx, err := findInPort(nameHint)
	if err != nil {
		log.Fatalf("could not find Blofeld MIDI in port: %v", err)
	}

	blo, closer, err := OpenBlofeld(blofeldDeviceID, portIdx)
	if err != nil {
		log.Fatalf("failed to open Blofeld output: %v", err)
	}
	defer closer()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "play":
			playTestNotes(blo, blofeldChannel)
			return
		case "single":
			singleTest(inPortIdx, portIdx, blo, blofeldChannel)
			return
		case "get":
			getPatch(inPortIdx, portIdx, blo, blofeldChannel)
			return
		case "set":
			setPatch(inPortIdx, portIdx, blo, blofeldChannel, blofeldDeviceID)
			return

		case "mcp":
			runMCP(inPortIdx, portIdx, blo, blofeldChannel)
			return

		default:
			log.Fatalf("unknown command %q", os.Args[1])
		}
	}
	log.Println("exiting: no command specified")
}

func findOutPort(nameFragment string) (int, error) {
	outs := midi.GetOutPorts()
	if len(outs) == 0 {
		return -1, fmt.Errorf("no MIDI outputs available")
	}

	lower := strings.ToLower(nameFragment)
	for _, out := range outs {
		if strings.Contains(strings.ToLower(out.String()), lower) {
			return out.Number(), nil
		}
	}

	return -1, fmt.Errorf("no MIDI output contains %q", nameFragment)
}

func findInPort(nameFragment string) (int, error) {
	ins := midi.GetInPorts()
	if len(ins) == 0 {
		return -1, fmt.Errorf("no MIDI inputs available")
	}

	lower := strings.ToLower(nameFragment)
	for _, in := range ins {
		if strings.Contains(strings.ToLower(in.String()), lower) {
			return in.Number(), nil
		}
	}

	return -1, fmt.Errorf("no MIDI input contains %q", nameFragment)
}

func bankToByte(bank string) (byte, error) {
	if bank == "" {
		return 0, errors.New("bank must not be empty")
	}
	ch := strings.ToUpper(bank)[0]
	if ch < 'A' || ch > 'H' {
		return 0, fmt.Errorf("bank must be A–H, got %q", bank)
	}
	return byte(ch - 'A'), nil
}
