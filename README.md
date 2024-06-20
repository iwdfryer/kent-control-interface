### Kent Control Interface

This repository contains our control interface for embedded dispenser, pass and frame devices. Users can send commands directly to the devices in order to change factory settings, scale calibrations, dispenser tuning and more. They system is in two parts, firstly the webUI opened in the users browser, this is the control panel for the user which will send messages to the second system part, ws-kent, which translates messages from the webUI to kent and forwards them to the embedded device specified. 

#### Setup 
- You will need a STM32F4 Olimex-E407 board flashed with [stm32-freertos-firmware](https://gitlab.com/karakuritech/dk/stm32-freertos-firmware) with internet connection.
- You will need to ensure your computer is connected to the same network as the board.

#### User Instructions

To use this tool simply navigate to the [webUI](http://karakuritech.gitlab.io/machine-testing/kent-control-interface/6605f7d0-d7d5-40ba-8414-a5da59291e59/) in your browser, and run the ws-kent binary in terminal with `./ws-kent`. 
Use examples and additional documentation can be found [here](https://karakuritech.atlassian.net/wiki/spaces/SW/pages/730562561/Kent+Control+Interface+webUI).

#### Developer Instructions

If you would like to make any changes to the webUI or binary file you will need to:
- Install Go, instructions can be found [here](https://golang.org/doc/install).
- Clone [dk-srv](https://gitlab.com/karakuritech/dk/dk-srv) in your GOPATH as it is a dependency. 
- Use the Makefile provided for the remaining setup, simply run `make`. 

##### Makefile Options
`make setup` to create a symlink to the dk-srv internal directory and fetch requiered files.\
`make binaries` to generate binaries for project files.\
`make clean` to remove all binaries, symlinks and fetched files.