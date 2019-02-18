# uptop [![Build Status](https://travis-ci.org/rothwerx/uptop.svg?branch=master)](https://travis-ci.org/rothwerx/uptop)
> Top-like tool for displaying per-process USS (unique set size) and PSS (proportional set size) memory usage

Standard Linux memory observation tools will show you the resident set size (RSS or RES) and virtual memory size (VMSIZE or VIRT) for each process, but neither are useful for giving you actual memory usage. RSS is more useful than VMSIZE as a general measurement, but RSS still includes the full memory consumed by each shared library. Here's a simplistic example: if you have 10 processes that each use 20kB of memory, and they all use the library foo.so which consumes 10kB of memory, each process will report 30kB memory usage even though foo.so only takes 10kB total.

Proportional set size (PSS) will split the size of the shared libraries between all processes using them. So in the example above, the 10kB of memory foo.so uses would be split between the 10 processes using it and the PSS for each of these processes would be 21kB instead of 30kB reported by RSS. Be aware that when watching this measurement, an increase or a decrease of the PSS value doesn't necessarily mean the process has allocated or freed memory - it could just be that the number of processes using the same shared libraries have changed.

Unique set size (USS) is the amount of memory that is unique to a process, i.e. not including shared libraries. It can be considered the memory that would be reclaimed should the process die. Of course this last statement assumes the process was either not using any shared library, or wasn't the last one to be using a shared library when it died.

## Installation

Until there's an official release, installation means building the (standalone) binary. See Development setup below. NOTE: this is Linux-only, and kernel version >= 2.6.27 at that.

## Usage example

Simply ensure uptop is executable, then run it in a terminal `./uptop`. Run with sudo to get more than just your own user processes.

You can quit with 'q' or Ctrl-c. While `uptop` is running, 'p' will sort by PSS, 'u' will sort by USS, 'r' will sort by RSS, 's' will sort by SwapPSS, and 'n' will sort by process name.

## Development setup

* Install Go (requires >= 1.11) [https://golang.org/doc/install](https://golang.org/doc/install)
* `go get -u github.com/rothwerx/uptop`

Since Go 1.11 introduced module versioning you'll want to enable the transitional variable to get termui. Change to the uptop source directory and run:
`GO111MODULE=on go get -u github.com/gizak/termui@master`
Older releases of termui will cause you headaches, and until termui stabilizes, termui@master will probably cause headaches too.


## Meta

Jeremiah Roth – [@rothwerx](https://twitter.com/rothwerx)

Distributed under the MIT license. See ``LICENSE`` for more information.

[https://github.com/rothwerx/uptop](https://github.com/rothwerx/)


## Contributing

1. Fork it (<https://github.com/rothwerx/uptop/fork>)
2. Create your feature branch (`git checkout -b feature/fooBar`)
3. Commit your changes (`git commit -am 'Add some fooBar'`)
4. Push to the branch (`git push origin feature/fooBar`)
5. Create a new Pull Request

## TODO
* Test on bigger iron (only tested on an AWS t2.nano instance)
* Non-interactive mode, e.g. allow user to specify -p <pid> for a specific process
* Static help bar with hotkey hints, or help overlay
* Why does `--once` print extra newlines
