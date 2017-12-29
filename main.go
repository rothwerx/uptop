package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
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
			cmdline := getCmdline(fname)
			if cmdline != "" {
				p := &Process{Basepath: fname, Command: cmdline}
				if err := p.PopulateInfo(); err != nil {
					fmt.Printf("error in PopulateInfo [%v]\n", err)
				}
				box.Items = append(box.Items, p)
			}
		}
	}
	return box
}

func main() {
	start := time.Now()
	procs := GetProcesses("/proc")

	// Print header and then the contents of each Process
	fmt.Printf("%6s  %-16s %-14s %5s  %5s  %5s  %5s  %-80s\n",
		"PID", "Name", "User", "Swap", "USS", "PSS", "RSS", "Command")
	for _, p := range procs.Items {
		fmt.Printf("%6d  %-16s %-14s %5d  %5d  %5d  %5d  %-80s\n",
			p.PID, p.Name, p.User, p.Swap, p.USS, p.PSS, p.RSS, p.Command)
	}
	elapsed := time.Since(start)
	fmt.Printf("Took %s to collect\n", elapsed)
}
