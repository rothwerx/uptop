package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	ui "github.com/gizak/termui"
)

// Process holds information about a process
type Process struct {
	Basepath            string
	PID                 int
	Name, User, Command string
	RSS, PSS, USS, Swap int
}

// AllProcs is a container for Processes
type AllProcs struct {
	Items []*Process
}

// UID->username map cache
var ucache = make(map[uint32]string)

var namergx = regexp.MustCompile(`\((.*?)\)`)

// scrapeSmaps sums select memory fields from /proc/<int>/smaps
func (p *Process) scrapeSmaps() error {
	if p.Basepath == "" {
		return fmt.Errorf("no path for Process")
	}
	leaf := filepath.Base(p.Basepath)
	pid, err := strconv.Atoi(leaf)
	if err != nil {
		return err
	}
	p.PID = pid
	smap := filepath.Join(p.Basepath, "smaps")
	file, err := os.Open(smap)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		p.RSS += getSmapMem(line, "Rss")
		p.PSS += getSmapMem(line, "Pss")
		p.Swap += getSmapMem(line, "Swap")
		p.USS += getSmapMem(line, "Private_Clean")
		p.USS += getSmapMem(line, "Private_Dirty")
	}
	return nil
}

// PopulateInfo fills in the Process attributes
func (p *Process) PopulateInfo() error {
	if err := p.scrapeSmaps(); err != nil {
		return err
	}
	p.Name = getProcName(p.Basepath)
	user, err := lookupUsername(p.Basepath)
	if err != nil {
		return err
	}
	p.User = user
	return nil
}

// lookupUsername looks up username for uid if not already in cache
func lookupUsername(file string) (string, error) {
	fileInfo, err := os.Stat(file)
	if err != nil {
		return "", err
	}
	uid := fileInfo.Sys().(*syscall.Stat_t).Uid
	if ucache[uid] != "" {
		return ucache[uid], nil
	}
	u, err := user.LookupId(strconv.FormatUint(uint64(uid), 10))
	if err != nil {
		return "", err
	}
	ucache[uid] = u.Username
	return u.Username, nil
}

// getSmapMem is a helper function to read partictular values from smaps
func getSmapMem(line, mment string) int {
	if strings.HasPrefix(line, mment) {
		f, err := strconv.Atoi(strings.Fields(line)[1])
		if err != nil {
			return 0
		}
		return f
	}
	return 0
}

// Determine if it's a process dir by checking if the dirname is an int
func isProc(path string) bool {
	basename := filepath.Base(path)
	_, err := strconv.Atoi(basename)
	if err != nil {
		return false
	}
	return true
}

// Get process name from stat file
func getProcName(path string) string {
	namepath := filepath.Join(path, "stat")
	statp, err := ioutil.ReadFile(namepath)
	if err != nil {
		fmt.Printf("error reading stat [%v]\n", err)
	}
	match := namergx.FindStringSubmatch(string(statp))
	return match[1]
}

// Returns the process cmdline
func getCmdline(path string) string {
	cmdpath := filepath.Join(path, "cmdline")
	cmdline, err := ioutil.ReadFile(cmdpath)
	if err != nil {
		fmt.Printf("read error [%v]\n", err)
	}
	cmdstring := string(cmdline)
	return strings.Replace(cmdstring, "\x00", " ", -1)
}

// GetProcesses returns a collection of Processes
func GetProcesses(rootpath string) AllProcs {
	box := AllProcs{}
	dirs, err := ioutil.ReadDir(rootpath)
	if err != nil {
		fmt.Printf("readdir error [%v]\n", err)
	}
	for _, f := range dirs {
		fname := filepath.Join(rootpath, f.Name())
		if isProc(fname) {
			p, ok := processIt(fname)
			if ok {
				box.Items = append(box.Items, p)
			}
		}
	}
	return box
}

// processIt returns a populated Process
func processIt(fpath string) (*Process, bool) {
	cmdline := getCmdline(fpath)
	if cmdline != "" {
		p := &Process{Basepath: fpath, Command: cmdline}
		if err := p.PopulateInfo(); err != nil {
			return nil, false
		}
		return p, true
	}
	return nil, false
}

func printProcesses(a AllProcs) {
	// Print header and then the contents of each Process
	fmt.Printf("%6s  %-16s %-14s %5s  %5s  %5s  %5s  %-80s\n",
		"PID", "Name", "User", "Swap", "USS", "PSS", "RSS", "Command")
	for _, p := range a.Items {
		fmt.Printf("%6d  %-16s %-14s %5d  %5d  %5d  %5d  %-80s\n",
			p.PID, p.Name, p.User, p.Swap, p.USS, p.PSS, p.RSS, p.Command)
	}
}

func tableFormat(a AllProcs) [][]string {
	tab := [][]string{[]string{"PID", "Name", "User", "Swap", "USS", "PSS", "RSS", "Command"},
		[]string{"---", "----", "----", "----", "---", "---", "---", "-------"}}
	for _, p := range a.Items {
		tab = append(tab, []string{strconv.Itoa(p.PID), p.Name, p.User, strconv.Itoa(p.Swap),
			strconv.Itoa(p.USS), strconv.Itoa(p.PSS), strconv.Itoa(p.RSS), p.Command})
	}
	return tab
}

func runTermui() {
	if err := ui.Init(); err != nil {
		log.Fatalln("cannot initialize termui")
	}
	defer ui.Close()

	procs := GetProcesses("/proc")

	tb := ui.NewTable()
	tb.Rows = tableFormat(procs)
	tb.Y = 0
	tb.X = 0
	tb.Separator = false
	tb.Border = false
	tb.Height = ui.TermHeight()
	tb.Width = ui.TermWidth()

	ui.Render(tb)

	// When the window resizes, the grid must adopt to the new size.
	ui.Handle("/sys/wnd/resize", func(ui.Event) {
		// Update the heights of list box and output box.
		tb.Height = ui.TermHeight()
		ui.Body.Width = ui.TermWidth()
		ui.Body.Align()
		ui.Render(tb)
	})

	// "q" or Ctrl-c stops the event loop.
	ui.Handle("/sys/kbd/q", func(ui.Event) {
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/C-c", func(ui.Event) {
		ui.StopLoop()
	})

	// Effective 2s refresh
	ui.Handle("/timer/1s", func(e ui.Event) {
		t := e.Data.(ui.EvtTimer)
		if t.Count%2 == 0 {
			procs := GetProcesses("/proc")
			tb.Rows = tableFormat(procs)
			ui.Render(tb)
		}
	})

	ui.Handle("/sys/kbd/r", func(ui.Event) {
		procs := GetProcesses("/proc")
		tb.Rows = tableFormat(procs)
		ui.Render(tb)
	})

	// start the event loop.
	ui.Loop()
}

func main() {
	runTermui()
}
