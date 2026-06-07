# RingWatch

A fast, beautiful terminal UI that reads your **Elden Ring** save file and shows,
at a glance, which bosses you still have left to fell — across the base game and
*Shadow of the Erdtree*. It watches your save and updates live as you play.

Pure Go, Linux-first. Built with [Bubble Tea v2](https://charm.land) and
[Lip Gloss v2](https://charm.land).

> **Inspired by** [RysanekDavid/The-Tarnished-Chronicle](https://github.com/RysanekDavid/The-Tarnished-Chronicle).
> RingWatch is an independent Go reimagining focused on a fast Linux TUI for boss
> tracking — no external tools, no OBS/overlay.

```
  ✦  RINGWATCH  ✦                                                    ● watching
  Davosso  ·  Lv 201  ·  71h 46m                  ██████░░░░░░░░░░  61/208  29%
 ──────────────────────────────────────────────────────────────────────────────
 ╭──────────────────────────────────╮╭──────────────────────────────────────────╮
 │ EARLY GAME · LEVELS 1–40 ─────────││ Limgrave                          7/20    │
 │  • Limgrave                  7/20 ││ ─────────────────────────────────────────│
 │  • Weeping Peninsula         1/10 ││  ✓ Grafted Scion                          │
 │  ✓ Stormveil Castle           2/2 ││  ○ Soldier of Godrick           ↗ (wiki)  │
 │ MID GAME · LEVELS 40–90 ──────────││  ✓ Tree Sentinel                          │
 │  • Liurnia of the Lakes      7/25 ││  ✓ Night's Cavalry                        │
 │  • Caelid                   2/14  ││  ○ Crucible Knight                        │
 ╰──────────────────────────────────╯╰──────────────────────────────────────────╯
  ↑↓ move · tab switch pane · / search · enter open wiki · c character · d dlc · q quit
```

## Features

- **Reads your real save** — parses the `ER0000.sl2` BND4 container directly (no
  external tools, no Wine). Elden Ring PC saves are *not* encrypted.
- **All 208 bosses** with a health bar, grouped by region in progression order,
  split into Early / Mid / Late game and the DLC.
- **Live updates** — watches the save file and shows a toast the moment a boss
  falls; no need to restart.
- **Multiple characters** — pick any of your save slots.
- **Search** any boss by name across every region.
- **Clickable boss links** — each boss links to its Fextralife wiki page via
  OSC 8 terminal hyperlinks (click, or press `enter` to open in your browser).
- **Auto-detects** your save across all Steam libraries (parses
  `libraryfolders.vdf`, supports non-default libraries and Flatpak Steam).

## Install

Requires Go 1.26+.

```sh
go build -o ringwatch ./cmd/ringwatch
# or install to $GOBIN:
go install github.com/Cidan/RingWatch/cmd/ringwatch@latest
```

## Usage

```sh
ringwatch                      # auto-detect the save, track the first character, watch live
ringwatch --slot 2             # track save slot index 2
ringwatch --save /path/ER0000.sl2   # use a specific save file (.sl2 or .co2)
ringwatch --no-watch           # don't watch for changes
```

### Keys

| Key | Action |
| --- | --- |
| `↑`/`↓` or `k`/`j` | Move selection |
| `tab` / `←` `→` | Switch between the Regions and Bosses panes |
| `/` | Search bosses by name (across all regions) |
| `enter` / `o` | Open the selected boss's wiki page |
| `c` | Choose a different character |
| `d` | Toggle DLC regions |
| `esc` | Clear the active search |
| `q` / `ctrl+c` | Quit |

## How it works

Elden Ring PC saves are plaintext **BND4** containers (the only integrity check
is a per-section MD5 — there is no AES encryption, contrary to popular belief).
RingWatch:

1. Reads the profile summaries (`user_data_10`) for the character list
   (name, level, playtime, occupied slots).
2. Walks each character slot to locate its event-flags blob, whose offset is
   variable (several preceding fields are data-dependent).
3. Maps each boss's event-flag id to a bit in that blob using the block→group
   table from [ClayAmore/ER-Save-Lib](https://github.com/ClayAmore/ER-Save-Lib).

## Layout

```
cmd/ringwatch      entry point + flags
internal/save      BND4 parsing, profiles, event-flag lookup
internal/tracker   boss dataset (embedded) + progress computation
internal/config    Steam save-path auto-detection
internal/watcher   fsnotify-based live watcher
internal/locations Fextralife links + OSC 8 hyperlinks
internal/ui        Bubble Tea v2 / Lip Gloss v2 TUI
```

The `save` and `tracker` packages have no UI dependencies, so item tracking can
be added later as a sibling dataset and a new tab.

## Credits & data

- Inspired by [The Tarnished Chronicle](https://github.com/RysanekDavid/The-Tarnished-Chronicle);
  boss → event-flag data was assembled from its dataset and cross-checked against
  Paramdex, ERTools, and other community sources.
- Event-flag location table from [ER-Save-Lib](https://github.com/ClayAmore/ER-Save-Lib).
- Boss links to the [Elden Ring Fextralife wiki](https://eldenring.wiki.fextralife.com).

## Roadmap

- Item / collectible tracking (next).
- Per-boss location overrides for any mismatched wiki links.

Not affiliated with FromSoftware or Bandai Namco.
