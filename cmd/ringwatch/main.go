// Command ringwatch is RingWatch: a TUI that tracks Elden Ring boss progress by
// reading and watching your save file.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Cidan/RingWatch/internal/config"
	"github.com/Cidan/RingWatch/internal/ui"
)

func main() {
	savePath := flag.String("save", "", "path to ER0000.sl2/.co2 (default: autodetect)")
	slot := flag.Int("slot", -1, "character slot index to track (default: first active)")
	noWatch := flag.Bool("no-watch", false, "disable live watching of the save file")
	flag.Parse()

	path := *savePath
	if path == "" {
		saves := config.DetectSaves()
		switch len(saves) {
		case 0:
			fmt.Fprintln(os.Stderr, "ringwatch: no Elden Ring save found automatically.")
			fmt.Fprintln(os.Stderr, "           Pass one explicitly:  ringwatch --save /path/to/ER0000.sl2")
			os.Exit(1)
		case 1:
			path = saves[0].Path
		default:
			path = saves[0].Path
			fmt.Fprintf(os.Stderr, "ringwatch: found %d saves; using %s\n           (choose another with --save)\n", len(saves), path)
		}
	}

	if err := ui.Run(ui.Options{SavePath: path, Slot: *slot, Watch: !*noWatch}); err != nil {
		fmt.Fprintf(os.Stderr, "ringwatch: %v\n", err)
		os.Exit(1)
	}
}
