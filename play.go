package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"gitlab.com/gomidi/midi/v2"
)

func singleTest(inPortIdx int, portIdx int, blo *Blofeld, blofeldChannel uint8) {

	if err := playTestNotes(blo, blofeldChannel); err != nil {
		log.Fatalf("failed to play test notes: %v", err)
	}

	log.Printf("Connected to Blofeld on port index %d (channel %d).\n", portIdx, blofeldChannel+1)

	var Bank = "H"
	var Program = 128

	p, devID, err := blo.RequestPatchDump(midi.GetInPorts()[inPortIdx], Bank, Program)
	if err != nil {
		log.Fatalf("failed to read patch: %v", err)
	}
	log.Println("Patch name", p.Name)
	log.Printf("Read patch from Bank %s, Program %d (device 0x%02X): %+v\n", Bank, Program, devID, p)

	asJson, err := json.MarshalIndent(&p, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal patch to JSON: %v", err)
	}
	if asJsonErr := json.Unmarshal(asJson, &p); asJsonErr != nil {
		log.Fatalf("JSON roundtrip failed: %v", asJsonErr)
	}

	p.Oscillators[0].Shape = p.Oscillators[0].Shape + 1

	log.Printf("Patch as JSON:\n%s\n", asJson)

	if err := blo.SendPatch(Bank, Program, p, devID); err != nil {
		log.Fatalf("failed to send patch: %v", err)
	}

	if err := playTestNotes(blo, blofeldChannel); err != nil {
		log.Fatalf("failed to play test notes: %v", err)
	}

	log.Println("Reading again.")
	p2, devID, err := blo.RequestPatchDump(midi.GetInPorts()[inPortIdx], Bank, Program)
	if err != nil {
		log.Fatalf("failed to read patch: %v", err)
	}
	log.Println("Patch name", p2.Name)

	asJson2, err := json.MarshalIndent(&p2, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal patch to JSON: %v", err)
	}
	log.Printf("Patch as JSON after sending modified version:\n%s\n", asJson2)

	fmt.Println("Done.")
}

func playTestNotes(blo *Blofeld, channel uint8) error {
	notes := []uint8{midi.C(4), midi.E(4), midi.G(4)}
	for _, n := range notes {
		if err := blo.Send(midi.NoteOn(channel, n, 100)); err != nil {
			return fmt.Errorf("note on failed for %d: %w", n, err)
		}
		time.Sleep(200 * time.Millisecond)
		if err := blo.Send(midi.NoteOff(channel, n)); err != nil {
			return fmt.Errorf("note off failed for %d: %w", n, err)
		}
	}
	return nil
}
