package calculator

import (
	"dca-platform/pkg/finance"
	"fmt"
	"time"
)

// StrategyResult armazena o resultado de uma estratégia
type StrategyResult struct {
	StrategyName    string
	TotalInvested   float64
	FinalValue      float64
	ReturnPercent   float64
	TotalAccumulated float64 // Qtd de ativo (BTC, Ouro onças, etc)
}

// Frequency define a frequência de investimento
type Frequency string

const (
	Daily   Frequency = "daily"
	Weekly  Frequency = "weekly"
	Monthly Frequency = "monthly"
)

// CalculateDCA calcula o retorno de uma estratégia DCA
func CalculateDCA(quotes []finance.Quote, initialAmount float64, amountPerPeriod float64, freq Frequency) StrategyResult {
	var totalInvested float64
	var totalAccumulated float64
	
	if len(quotes) == 0 {
		return StrategyResult{StrategyName: "DCA Bitcoin (Sem dados)"}
	}

	// Compra Inicial (Lump Sum parcial)
	if initialAmount > 0 {
		bought := initialAmount / quotes[0].Close
		totalAccumulated += bought
		totalInvested += initialAmount
	}

	lastPurchaseDate := time.Time{}

	// Compra Recorrente (DCA)
	if amountPerPeriod > 0 {
		for _, q := range quotes {
			shouldBuy := false
	
			if lastPurchaseDate.IsZero() {
				shouldBuy = true
			} else {
				switch freq {
				case Daily:
					// Compra todo dia que tiver dados
					shouldBuy = true
				case Weekly:
					// Se passou 7 dias ou mais desde a ultima compra
					if q.Date.Sub(lastPurchaseDate).Hours() >= 24*7 {
						shouldBuy = true
					}
				case Monthly:
					// Se mudou o mês
					if q.Date.Month() != lastPurchaseDate.Month() || q.Date.Year() != lastPurchaseDate.Year() {
						shouldBuy = true
					}
				}
			}
	
			if shouldBuy {
				bought := amountPerPeriod / q.Close
				totalAccumulated += bought
				totalInvested += amountPerPeriod
				lastPurchaseDate = q.Date
			}
		}
	}
	
	// Valor final = acumulado * ultimo preço
	lastPrice := quotes[len(quotes)-1].Close
	finalValue := totalAccumulated * lastPrice
	
	ret := 0.0
	if totalInvested > 0 {
		ret = (finalValue - totalInvested) / totalInvested * 100
	}
	
	name := "DCA " + string(freq)
	if initialAmount > 0 && amountPerPeriod == 0 {
		name = "Investimento Único (Lump Sum)"
	} else if initialAmount > 0 {
		name = fmt.Sprintf("Híbrido (Ini: $%.0f + DCA)", initialAmount)
	}

	return StrategyResult{
		StrategyName:     name,
		TotalInvested:    totalInvested,
		FinalValue:       finalValue,
		ReturnPercent:    ret,
		TotalAccumulated: totalAccumulated,
	}
}

// CalculateLumpSum calcula o retorno de um investimento único no início
func CalculateLumpSum(quotes []finance.Quote, totalAmount float64, name string) StrategyResult {
	if len(quotes) == 0 {
		return StrategyResult{StrategyName: name + " (Sem dados)"}
	}

	firstPrice := quotes[0].Close
	lastPrice := quotes[len(quotes)-1].Close

	accumulated := totalAmount / firstPrice
	finalValue := accumulated * lastPrice

	ret := (finalValue - totalAmount) / totalAmount * 100

	return StrategyResult{
		StrategyName:     name,
		TotalInvested:    totalAmount,
		FinalValue:       finalValue,
		ReturnPercent:    ret,
		TotalAccumulated: accumulated,
	}
}
