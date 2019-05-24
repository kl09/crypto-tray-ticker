package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/getlantern/systray"
	"github.com/op/go-logging"
	"net"
	"net/http"
	"time"
)

const binanceSource = "binance"
const coincapSource = "coincap"

type Token struct {
	ID                string      `json:"id"`
	Symbol            string      `json:"symbol"`
	Name              string      `json:"name"`
	PriceUsd          json.Number `json:"priceUsd"`
	ChangePercent24Hr json.Number `json:"changePercent24Hr"`
	Comment           string      `json:"comment"`
}

type App struct {
	log           *logging.Logger
	client        *http.Client
	tokensLimit   int
	blinkOnUpdate bool
	updatesDelay  time.Duration
	fileName      string
}

func NewApp(tokensLimit int, blinkOnUpdate bool, updatesDelay time.Duration, fileName string) *App {
	return &App{
		log: logging.MustGetLogger(appName),
		client: &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout: 2 * time.Second,
				}).DialContext,
			},
			Timeout: 2 * time.Second,
		},
		tokensLimit:   tokensLimit,
		blinkOnUpdate: blinkOnUpdate,
		updatesDelay:  updatesDelay,
		fileName:      fileName,
	}
}

func (a *App) Start() {
	systray.Run(a.onReady, a.onExit)
}

func (a *App) onExit() {

}

func (a *App) onReady() {
	var (
		tokens []*Token
		err    error
	)
	source := make(chan string, 100)
	clickedTokens := make(chan []*Token, 100)

	// load saved tokens
	go load(a.fileName, clickedTokens, source)

	tokens, err = a.loadTokensForMenu()
	if err != nil {
		panic(err)
	}

	a.createMenu(tokens, clickedTokens, source)
	a.menuLoop(clickedTokens, source)
}

func (a *App) loadTokensForMenu() ([]*Token, error) {
	var err error

	for i := 0; i < 10; i++ {
		tokens, err := a.getTokens()

		if err == nil {
			a.log.Info("tokens loaded successful")
			return tokens, err
		}

		a.log.Error("can't get tokens at start", err)
		systray.SetTitle(appName + ": " + err.Error())
		time.Sleep(2 * time.Second)
	}

	return nil, err
}

func (a *App) createMenu(tokens []*Token, clickedTokens chan []*Token, source chan string) {
	menuToken := make(map[*systray.MenuItem]*Token)
	menuSource := make(map[*systray.MenuItem]string)

	// create menu item for each token
	for _, token := range tokens {
		menuItem := systray.AddMenuItem(token.Symbol, token.Name)
		menuToken[menuItem] = token

		go func() {
			for {
				select {
				case <-menuItem.ClickedCh:
					clickedTokens <- []*Token{menuToken[menuItem]}
				}
			}
		}()
	}

	systray.AddSeparator()

	for _, sourceName := range []string{coincapSource, binanceSource} {
		menuItem := systray.AddMenuItem(sourceName, "")
		menuSource[menuItem] = sourceName

		go func() {
			for {
				select {
				case <-menuItem.ClickedCh:
					source <- menuSource[menuItem]
				}
			}
		}()
	}

	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Close the app")

	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
	}()
}

func (a *App) menuLoop(chClickedTokens chan []*Token, chSource chan string) {
	selected := make([]*Token, 0)
	var source string

	// wait for selected tokens or new source
	go func() {
		for {
			select {
			case clickedTokens := <-chClickedTokens:

				for _, clickedTokenID := range clickedTokens {
					found := false
					for i, v := range selected {
						if v.ID == clickedTokenID.ID { // already selected - delete it
							found = true
							selected = append(selected[:i], selected[i+1:]...)
						}
					}
					if !found {
						selected = append(selected, clickedTokenID)
					}
				}

				a.updateTray(selected, source)
				go save(a.fileName, selected, source)
			case source = <-chSource:
				a.updateTray(selected, source)
				go save(a.fileName, selected, source)
			}
		}
	}()

	// Tokens periodic update
	go func() {
		for {
			a.updateTray(selected, source)
			time.Sleep(a.updatesDelay)
		}
	}()
}

func (a *App) updateTray(selected []*Token, source string) {
	var (
		trayTitle      string
		price, percent float64
	)

	if source != "" {
		trayTitle = source + ": "
	} else {
		trayTitle = coincapSource + ": "
	}

	if len(selected) > 0 {
		for _, token := range selected {
			tokenUpdated, err := a.getToken(token, source)

			if err != nil {
				a.log.Error("can't get token", err)
				trayTitle += fmt.Sprintf("%s - %s ", token.Symbol, err.Error())
			} else if tokenUpdated != nil {
				price, _ = tokenUpdated.PriceUsd.Float64()
				trayTitle += fmt.Sprintf("%s - %.3f$ ", tokenUpdated.Symbol, price)

				percent, _ = tokenUpdated.ChangePercent24Hr.Float64()
				if percent != 0 {
					trayTitle += fmt.Sprintf("[%.2f%%]  ", percent)
				}
			}
		}
	} else {
		trayTitle += "select coin"
	}

	if a.blinkOnUpdate {
		systray.SetTitle("")
		time.Sleep(100 * time.Millisecond)
	}

	systray.SetTitle(trayTitle)
}

func (a *App) getToken(token *Token, source string) (*Token, error) {
	var err error

	switch source {
	case binanceSource:
		token, err = a.getBinanceToken(token, "USDT")
	case coincapSource:
		token, err = a.getCoinCapToken(token)
	default:
		token, err = a.getCoinCapToken(token)
	}

	return token, err
}

func (a *App) getCoinCapToken(token *Token) (*Token, error) {
	type Result struct {
		Data      *Token `json:"data"`
		Timestamp int    `json:"timestamp"`
	}
	result := Result{}
	url := fmt.Sprintf("https://api.coincap.io/v2/assets/%s", token.ID)

	r, err := makeRequest(context.Background(), a.client, "GET", url, nil, nil)
	if err != nil {
		a.log.Error(err)
		return nil, fmt.Errorf("connection problem")
	}

	defer func() {
		err = r.Body.Close()
		if err != nil {
			a.log.Error(err)
		}
	}()

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server is not available")
	}

	err = json.NewDecoder(r.Body).Decode(&result)
	if err != nil {
		a.log.Error(err)
		return nil, fmt.Errorf("parsing problem")
	}

	return result.Data, nil
}

func (a *App) getBinanceToken(token *Token, currency string) (*Token, error) {
	type Result struct {
		Price json.Number `json:"price"`
		Mins  int         `json:"mins"`
	}
	result := Result{}
	url := fmt.Sprintf("https://api.binance.com/api/v3/avgPrice?symbol=%s%s", token.Symbol, currency)

	r, err := makeRequest(context.Background(), a.client, "GET", url, nil, nil)
	if err != nil {
		a.log.Error(err)
		return nil, fmt.Errorf("connection problem")
	}

	defer func() {
		err = r.Body.Close()
		if err != nil {
			a.log.Error(err)
		}
	}()

	if r.StatusCode == http.StatusBadRequest {
		return token, fmt.Errorf("token not found")
	}

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server is not available")
	}

	err = json.NewDecoder(r.Body).Decode(&result)
	if err != nil {
		a.log.Error(err)
		return nil, fmt.Errorf("parsing problem")
	}

	token.PriceUsd = result.Price
	token.ChangePercent24Hr = ""

	return token, nil
}

func (a *App) getTokens() ([]*Token, error) {
	type Result struct {
		Data      []*Token `json:"data"`
		Timestamp int      `json:"timestamp"`
	}
	result := Result{}

	url := fmt.Sprintf("https://api.coincap.io/v2/assets?limit=%d", a.tokensLimit)

	r, err := makeRequest(context.Background(), a.client, "GET", url, nil, nil)
	if err != nil {
		a.log.Error(err)
		return nil, fmt.Errorf("connection problem")
	}

	defer func() {
		err = r.Body.Close()
		if err != nil {
			a.log.Error(err)
		}
	}()

	if r.StatusCode != http.StatusOK {
		a.log.Error(err)
		return nil, fmt.Errorf("server is not available")
	}

	err = json.NewDecoder(r.Body).Decode(&result)
	if err != nil {
		a.log.Error(err)
		return nil, fmt.Errorf("parsing problem")
	}

	return result.Data, nil
}
