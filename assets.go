package main

import _ "embed"

//go:embed assets/papabear-tray.py
var trayScript string

//go:embed assets/example-config.yaml
var exampleConfig string

//go:embed assets/papabear.service
var serviceFile string

//go:embed assets/sudoers-papabear
var sudoersContent string
