# Blofeld MCP

Go utilities and an MCP server for the Waldorf Blofeld synthesizer. The tools find your Blofeld MIDI ports, dump and send patches, and expose note/patch actions to Model Context Protocol clients.

- SysEx reference: `waldorf_blofeld_sysex_documentation_v.1.04.txt` (share the URL to this file when chatting with an AI so it can cite the spec).
- Synth basics: 3 oscillators with wavetables/VA shapes, dual multi-mode filters, 3 envelopes, 3 LFOs, 16-slot mod matrix, arpeggiator, and two FX. Desktop, monotimbral per MIDI channel; factory listens on channel 5 (0-based 4).

## Quick start
- Build: `go build -o blofeldmcp .`
- Run MCP server (stdio): `./blofeldmcp mcp`

## Using with AI chats
- Claude: add an MCP server entry that runs `./blofeldmcp mcp`; Claude can call `blofeld_describe-sysex`, `blofeld_get-patch`, `blofeld_send-patch`, and note-play tools.
- ChatGPT MCP: add a custom MCP server pointing to `./blofeldmcp mcp`; grant MIDI access when prompted and let the model call the tools.

## Debug helpers
- Test notes: `./blofeldmcp play`
- Single sound test: `./blofeldmcp single`
- Dump a patch: `./blofeldmcp get`
- Load a patch: `./blofeldmcp set`


