package main

import (
	"flag"
	"time"
)

var appName = "crypto-tray-ticker"

var (
	blinkOnUpdate = flag.Bool("blink", false, "blink on update")
	updatesDelay  = flag.Duration("updates-delay", 5*time.Second, "Delay between updates")
	tokensLimit   = flag.Int("tokens-limit", 20, "limit of tokens in menu")
	fileName      = flag.String("file-name", "selected-tokens.json", "name of file to save selected tokens")
)

func main() {
	flag.Parse()

	app := NewApp(*tokensLimit, *blinkOnUpdate, *updatesDelay, *fileName)
	app.Start()
}
