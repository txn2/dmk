package tui

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/jroimartin/gocui"
	"github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

var yellow = color.New(color.FgYellow).SprintFunc()
var green = color.New(color.FgGreen).SprintFunc()
var grey = color.New(color.FgBlack).SprintFunc()

// GuiDataCfg
type GuiDataCfg struct {
	MachineName string
	Fl          *os.File
	Quiet       bool
	Done        bool
}

// guiData
type guiData struct {
	cfg       *GuiDataCfg
	ui        *gocui.Gui
	statusOut []string
	total     int
	count     int
	d         time.Duration
	start     time.Time
	writes    int
	prevWrite []byte
}

// NewGui creates a CLI GUI for migration data
func NewGui(cfg *GuiDataCfg) (*guiData, *sync.WaitGroup) {
	g, err := gocui.NewGui(gocui.Output256)
	if err != nil {
		log.Panicln(err)
	}

	gui := &guiData{
		cfg:       cfg,
		ui:        g,
		statusOut: []string{},
		total:     0,
		count:     0,
		writes:    0,
		start:     time.Now(),
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func(group *sync.WaitGroup) {
		defer g.Close()

		g.SetManagerFunc(guiLayout)

		if err := g.SetKeybinding("", gocui.KeyCtrlQ, gocui.ModNone, guiQuit); err != nil {
			log.Panicln(err)
		}

		if cfg.Quiet {
			g.Update(func(gui *gocui.Gui) error {
				viewSo, err := gui.View("script_out")
				if err != nil {
					panic(err)
				}

				viewSl, err := gui.View("status_log")
				if err != nil {
					panic(err)
				}

				viewSts, err := gui.View("stats")
				if err != nil {
					panic(err)
				}

				fmt.Fprint(viewSo, grey(" Quiet mode."))
				fmt.Fprint(viewSl, grey(" Quiet mode."))
				fmt.Fprint(viewSts, grey("\n Quiet mode."))
				return nil
			})
		}

		if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
			log.Panicln(err)
		}

		group.Done()
	}(&wg)

	return gui, &wg
}

// guiLayout
func guiLayout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("status_log", 0, 0, maxX-1, maxY/3); err != nil {
		v.Autoscroll = true
		v.Title = "Migration Status"
		if err != gocui.ErrUnknownView {
			return err
		}
	}

	if v, err := g.SetView("help", 0, maxY-4, 20, maxY-1); err != nil {
		v.Title = "Help"

		fmt.Fprint(v, green(" Ctrl-q to quit."))
	}

	if v, err := g.SetView("stats", 0, maxY/3+1, 20, maxY-5); err != nil {
		v.Title = "Stats"
	}

	if v, err := g.SetView("script_out", 21, maxY/3+1, maxX-1, maxY-1); err != nil {
		v.Title = "Script Output"
		v.Autoscroll = true
		fmt.Fprint(v, grey(" Transformation script output."))
	}

	return nil
}

// guiQuit
func guiQuit(g *gocui.Gui, v *gocui.View) error {
	//g.Close()
	return gocui.ErrQuit
}

// StatusMessage
type StatusMessage struct {
	Level       string   `json:"level"`
	Ts          float64  `json:"ts"`
	Msg         string   `json:"msg"`
	Type        string   `json:"Type"`
	MachineName string   `json:"MachineName"`
	ScriptPrint string   `json:"ScriptPrint"`
	Args        []string `json:"Args"`
	Count       int      `json:"Count"`
}

// Write implements SyncWriter
func (g *guiData) Write(p []byte) (nn int, err error) {
	g.writes += 1

	if g.cfg.Fl != nil {
		g.cfg.Fl.Write(p)
	}

	sm := &StatusMessage{}
	err = json.Unmarshal(p, sm)
	if err != nil {
		panic(err)
	}

	if sm.Count > 0 {
		started := g.start
		t := time.Now()
		dur := t.Sub(started)
		g.count = sm.Count
		g.ui.Update(func(gui *gocui.Gui) error {
			v, err := gui.View("stats")
			if err != nil {
				// handle error
			}
			v.Clear()

			fmt.Fprintf(v, grey(" Processed:\n")+" %d", sm.Count)
			fmt.Fprintf(v, grey("\n Duration:\n")+" %v", dur)
			fmt.Fprintf(v, grey("\n Avg:\n")+" %v", dur/time.Duration(sm.Count))
			return nil
		})
	}

	if g.cfg.Quiet {
		// quiet mode does not print messages
		return
	}

	// add to buffer and update views
	g.ui.Update(func(gui *gocui.Gui) error {
		v, err := gui.View("status_log")
		if err != nil {
			// handle error
		}
		fmt.Fprintf(v, "\n %25s| %06d| %35s|"+grey(" Args:")+" %v", yellow(sm.Type), g.count, sm.Msg, sm.Args)
		return nil
	})

	if sm.Type == "ScriptOutput" {
		count := g.count
		g.ui.Update(func(gui *gocui.Gui) error {
			v, err := gui.View("script_out")
			if err != nil {
				// handle error
			}

			fmt.Fprintf(v, "\n %06d > %s", count+1, sm.ScriptPrint)
			return nil
		})
	}

	//log.Printf("Line: %s", p)
	return
}

// Sync implements SyncWriter
func (g *guiData) Sync() error {
	g.cfg.Fl.Sync()
	return nil
}
