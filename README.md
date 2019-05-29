# Crypto tray ticker

App for looking crypto prices in tray.

*You can look for prices of top coins from coincap.io and binance.com*
<b>Tested on Linux</b>

<img src="https://thumbs.gfycat.com/GrotesqueIncredibleCarpenterant-size_restricted.gif" width="600">

## Usage

```go
go get github.com/kl09/crypto-tray-ticker
go build
./crypto-tray-ticker
```

## Additional settings

1. Blink on update, default: false
```bash
./crypto-tray-ticker -blink=true
```

2. Delay between updates, default: 5s
```bash
./crypto-tray-ticker -updates-delay=30s
```

3. Limit of tokens in menu, default: 20
```bash
./crypto-tray-ticker -tokens-limit=40
```


## License
[The MIT License (MIT)](https://opensource.org/licenses/MIT)


Star ⭐⭐⭐ it if u like