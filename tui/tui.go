package tui

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/jroimartin/gocui"
)

// GuiDataCfg
type GuiDataCfg struct {
	MachineName string
	Fl          *os.File
}

// guiData
type guiData struct {
	cfg       *GuiDataCfg
	ui        *gocui.Gui
	statusOut []string
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
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func(group *sync.WaitGroup) {
		defer g.Close()

		g.SetManagerFunc(guiLayout)

		if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, guiQuit); err != nil {
			log.Panicln(err)
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
	if v, err := g.SetView("hello", 0, 0, maxX-10, maxY/2); err != nil {
		v.Autoscroll = true
		v.Title = "Migration Status"
		if err != gocui.ErrUnknownView {
			return err
		}
	}
	return nil
}

// guiQuit
func guiQuit(g *gocui.Gui, v *gocui.View) error {
	//g.Close()
	return gocui.ErrQuit
}

// Write implements SyncWriter
func (g *guiData) Write(p []byte) (nn int, err error) {

	if g.cfg.Fl != nil {
		g.cfg.Fl.Write(p)
	}

	g.statusOut = append(g.statusOut, string(p))
	status := g.statusOut

	// add to buffer and update views
	g.ui.Update(func(g *gocui.Gui) error {
		v, err := g.View("hello")
		if err != nil {
			// handle error
		}
		v.Clear()
		fmt.Fprint(v, status)
		return nil
	})

	//log.Printf("Line: %s", p)
	return
}

// Sync implements SyncWriter
func (g *guiData) Sync() error {
	g.cfg.Fl.Sync()
	return nil
}
