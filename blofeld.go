package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
)

const PatchSize = 383 // Expected SDATA payload size for a Blofeld patch

func dumpBytes(data []byte, file string) {

	/*
		date := time.Now().Format("20060102150405")

		f, err := os.Create("dump/" + date + "_" + file)
		if err != nil {
			log.Fatalf("failed to create dump file %s: %v", file, err)
		}
		defer f.Close()
	*/
	f := os.Stderr

	fmt.Fprintf(f, "Dumping %d bytes to %s:\n", len(data), file)

	for i, b := range data {
		fmt.Fprintf(f, "%d 0x%02X\n", i, b)
	}
}

type Oscillator struct {
	Shape    byte `json:"shape"`
	Pitch    byte `json:"pitch"`
	Detune   byte `json:"detune"`
	PW       byte `json:"pw"`
	PWM      byte `json:"pwm"`
	FM       byte `json:"fm"`
	FMSource byte `json:"fm_source"`
}

type Filter struct {
	Type   byte `json:"type"`
	Cutoff byte `json:"cutoff"`
	Res    byte `json:"res"`
	Drive  byte `json:"drive"`
	EnvAmt byte `json:"env_amt"`
}

type Envelope struct {
	Attack  byte `json:"attack"`
	Decay   byte `json:"decay"`
	Sustain byte `json:"sustain"`
	Release byte `json:"release"`
}

type LFO struct {
	Shape byte `json:"shape"`
	Speed byte `json:"speed"`
}

type Effect struct {
	Type   byte `json:"type"`
	Param1 byte `json:"param1"`
	Param2 byte `json:"param2"`
}

type ModulationMatrix struct {
	Source byte `json:"source"`
	Amount byte `json:"amount"`
	Dest   byte `json:"dest"`
}

type Patch struct {
	Oscillators [3]Oscillator `json:"oscillators"`

	Filters [2]Filter `json:"filters"`

	// Mixer
	MixOsc1  byte `json:"mix_osc1"`
	MixOsc2  byte `json:"mix_osc2"`
	MixOsc3  byte `json:"mix_osc3"`
	MixNoise byte `json:"mix_noise"`
	MixRing  byte `json:"mix_ring"`

	FilterRouting byte `json:"filter_routing"`

	Envelopes [3]Envelope `json:"envelopes"`

	LFOs [3]LFO `json:"lfos"`

	// Mod Matrix (16 slots × (source, amount, dest))
	ModMatrix [16]ModulationMatrix `json:"mod_matrix"`

	// Arpeggiator section
	ArpMode    byte `json:"arp_mode"`
	ArpPattern byte `json:"arp_pattern"`
	ArpSpeed   byte `json:"arp_speed"`
	ArpRange   byte `json:"arp_range"`
	ArpSwing   byte `json:"arp_swing"`
	ArpLength  byte `json:"arp_length"`
	ArpAccent  byte `json:"arp_accent"`

	// FX
	Effects [2]Effect `json:"effects"`

	// Amp
	AmpVolume byte `json:"amp_volume"`
	AmpPan    byte `json:"amp_pan"`
	AmpDrive  byte `json:"amp_drive"`

	// Master tuning + globals
	MasterTune byte `json:"master_tune"`

	// Patch name – 16 ASCII chars (363–378)
	Name string `json:"name"`

	// Category + Subcategory
	Category    byte `json:"category"`
	SubCategory byte `json:"subcategory"`

	// Arp micro settings
	ArpClock byte `json:"arp_clock"`
	ArpSort  byte `json:"arp_sort"`

	// Raw holds the original SDATA bytes so we can round-trip without losing unknown fields.
	Raw []byte `json:"-"`
}

var random = rand.New(rand.NewSource(time.Now().UnixNano()))
var oscFieldMapping = []struct {
	pitch    int
	detune   int
	fmSource int
	fm       int
	shape    int
	pw       int
	pwm      int
}{
	{pitch: 2, detune: 3, fmSource: 6, fm: 7, shape: 8, pw: 9, pwm: 11},
	{pitch: 18, detune: 19, fmSource: 22, fm: 23, shape: 24, pw: 25, pwm: 27},
	{pitch: 34, detune: 35, fmSource: 38, fm: 39, shape: 40, pw: 41, pwm: 43},
}

// RandomizeOscillators mutates only oscillator-related parameters using a PRNG.
func (p *Patch) RandomizeOscillators() {

	randByte := func() byte {
		return byte(random.Intn(128)) // constrain to 0–127 MIDI range
	}

	// Limit shape to the first 5 oscillator types (0–4), avoiding custom samples/wavetables.
	randShape := func() byte {
		return byte(random.Intn(5))
	}

	for i := range p.Oscillators {
		p.Oscillators[i].Shape = randShape()
		p.Oscillators[i].Pitch = randByte()
		p.Oscillators[i].Detune = randByte()
		p.Oscillators[i].PW = randByte()
		p.Oscillators[i].PWM = randByte()
		p.Oscillators[i].FM = randByte()
		p.Oscillators[i].FMSource = randByte()
	}
}

func ParseSDATA(data []byte) (*Patch, error) {
	if len(data) != PatchSize {
		return nil, errors.New("invalid SDATA length")
	}

	p := &Patch{
		Raw: append([]byte(nil), data...),
	}

	// Map oscillator fields per Blofeld spec indexes (3.1 SDATA table).
	for i, m := range oscFieldMapping {
		if i >= len(p.Oscillators) {
			break
		}
		p.Oscillators[i].Pitch = data[m.pitch]
		p.Oscillators[i].Detune = data[m.detune]
		p.Oscillators[i].FMSource = data[m.fmSource]
		p.Oscillators[i].FM = data[m.fm]
		p.Oscillators[i].Shape = data[m.shape]
		p.Oscillators[i].PW = data[m.pw]
		p.Oscillators[i].PWM = data[m.pwm]
	}

	// Keep some metadata handy.
	name := make([]byte, 16)
	copy(name, data[363:363+16])
	p.Name = string(bytes.TrimRight(name, "\x00"))
	p.Category = data[379]
	p.SubCategory = data[380]
	p.ArpClock = data[381]
	p.ArpSort = data[382]

	return p, nil
}

func (p *Patch) ToSDATA() ([]byte, error) {
	var data []byte
	if len(p.Raw) == PatchSize {
		data = append([]byte(nil), p.Raw...)
	} else {
		data = make([]byte, PatchSize)
	}

	// Map oscillator fields back to SDATA indexes (3.1 SDATA table).
	for i, m := range oscFieldMapping {
		if i >= len(p.Oscillators) {
			break
		}
		osc := p.Oscillators[i]
		data[m.pitch] = osc.Pitch
		data[m.detune] = osc.Detune
		data[m.fmSource] = osc.FMSource
		data[m.fm] = osc.FM
		data[m.shape] = osc.Shape
		data[m.pw] = osc.PW
		data[m.pwm] = osc.PWM
	}

	// Patch name and metadata preserved/overwritten as before.
	if p.Name != "" {
		name := []byte(p.Name)
		if len(name) > 16 {
			name = name[:16]
		}
		for len(name) < 16 {
			name = append(name, 0)
		}
		copy(data[363:363+16], name)
	}

	if p.Category != 0 {
		data[379] = p.Category
	}
	if p.SubCategory != 0 {
		data[380] = p.SubCategory
	}
	if p.ArpClock != 0 {
		data[381] = p.ArpClock
	}
	if p.ArpSort != 0 {
		data[382] = p.ArpSort
	}

	return data, nil
}

func (p *Patch) ToSNDD(deviceID byte, bank byte, program byte) ([]byte, error) {
	sdata, err := p.ToSDATA()

	dumpBytes(sdata, "sent_sdata.txt")

	if err != nil {
		return nil, err
	}

	out := []byte{0xF0, 0x3E, 0x13, deviceID, 0x10, bank, program}
	out = append(out, sdata...)

	var chk byte
	for _, b := range sdata {
		chk = (chk + b) & 0x7F
	}
	out = append(out, chk, 0xF7)

	dumpBytes(out, "sent_sndd.txt")

	return out, nil
}

type Blofeld struct {
	devID byte
	out   drivers.Out
}

func OpenBlofeld(devID byte, portIndex int) (*Blofeld, func(), error) {
	outs, err := drivers.Outs()
	if err != nil {
		return nil, nil, err
	}

	if portIndex < 0 || portIndex >= len(outs) {
		return nil, nil, fmt.Errorf("output port index %d out of range", portIndex)
	}

	out := outs[portIndex]
	if err := out.Open(); err != nil {
		return nil, nil, err
	}

	closer := func() {
		_ = out.Close()
		drivers.Close()
	}
	log.Println("Opened Blofeld MIDI output port", devID, out.String())
	return &Blofeld{
		devID: devID,
		out:   out,
	}, closer, nil
}

// Send transmits a MIDI message to the Blofeld output port.
func (b *Blofeld) Send(msg midi.Message) error {
	if !b.out.IsOpen() {
		if err := b.out.Open(); err != nil {
			return err
		}
	}
	return b.out.Send(msg.Bytes())
}

// SendSysEx transmits a raw SysEx payload.
func (b *Blofeld) SendSysEx(data []byte) error {
	return b.Send(midi.Message(data))
}

// RequestPatchDump asks Blofeld for a single program and waits for SNDD.
func (b *Blofeld) RequestPatchDump(inPort drivers.In, bank string, program int) (*Patch, byte, error) {
	bankByte, err := bankToByte(bank)
	if err != nil {
		return nil, 0, err
	}

	if program < 1 || program > 128 {
		return nil, 0, fmt.Errorf("program must be in range 1–128, got %d", program)
	}
	progByte := byte(program - 1) // Blofeld expects 0–127

	msgCh := make(chan midi.Message, 1)

	stop, err := midi.ListenTo(inPort, func(msg midi.Message, _ int32) {
		if len(msg) > 0 && msg[0] == 0xF0 {
			select {
			case msgCh <- msg:
			default:
			}
		}
	}, midi.UseSysEx(), midi.SysExBufferSize(2048))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to listen for patch dump: %w", err)
	}
	defer stop()

	log.Printf("Requesting patch dump from device ID 0x%02X", b.devID)
	req := []byte{0xF0, 0x3E, 0x13, b.devID, 0x00, bankByte, progByte, 0xF7}
	log.Println("Sending SysEx request")
	if err := b.SendSysEx(req); err != nil {
		return nil, 0, fmt.Errorf("failed to request patch dump: %w", err)
	}

	select {
	case msg := <-msgCh:
		log.Println("Received SysEx message")
		patch, respDevID, parseErr := parseSNDD(msg)
		if parseErr != nil {
			return nil, 0, parseErr
		}
		return patch, respDevID, nil
	case <-time.After(5 * time.Second):
		log.Println("Timed out waiting for patch dump")
	}

	return nil, 0, errors.New("timed out waiting for patch dump")
}

func parseSNDD(msg midi.Message) (*Patch, byte, error) {

	dumpBytes(msg, "received_sysex.txt")

	if len(msg) != PatchSize+9 {
		return nil, 0, fmt.Errorf("unexpected dump size %d (want %d)", len(msg), PatchSize+9)
	}

	if msg[0] != 0xF0 || msg[len(msg)-1] != 0xF7 {
		return nil, 0, errors.New("message is not a SysEx frame")
	}

	if msg[1] != 0x3E || msg[2] != 0x13 {
		return nil, 0, errors.New("not a Waldorf Blofeld SysEx")
	}

	if msg[4] != 0x10 {
		return nil, 0, fmt.Errorf("unexpected message type 0x%02X (expected SNDD 0x10)", msg[4])
	}

	sdata := msg[7 : 7+PatchSize]
	checksum := msg[7+PatchSize]

	var chk byte
	for _, b := range sdata {
		chk = (chk + b) & 0x7F
	}

	if checksum != 0x7F && chk != checksum {
		return nil, 0, fmt.Errorf("checksum mismatch: expected 0x%02X got 0x%02X", chk, checksum)
	}

	dumpBytes(sdata, "received_sdata.txt")

	patch, err := ParseSDATA(sdata)
	return patch, msg[3], err
}

// SendPatch transmits a patch to the given bank/program.
func (b *Blofeld) SendPatch(bank string, program int, p *Patch, devID byte) error {
	bankByte, err := bankToByte(bank)
	if err != nil {
		return err
	}
	if program < 1 || program > 128 {
		return fmt.Errorf("program must be in range 1–128, got %d", program)
	}
	progByte := byte(program - 1)

	payload, err := p.ToSNDD(devID, bankByte, progByte)
	if err != nil {
		return fmt.Errorf("failed to build SNDD payload: %w", err)
	}

	if err := b.SendSysEx(payload); err != nil {
		return fmt.Errorf("failed to send patch to bank %s program %d: %w", bank, program, err)
	}
	return nil
}
