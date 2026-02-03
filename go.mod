module ats

go 1.22

require (
	github.com/alpacahq/alpaca-trade-api-go/v3 v3.0.0
)

replace github.com/alpacahq/alpaca-trade-api-go/v3 => ./internal/alpaca_stub
