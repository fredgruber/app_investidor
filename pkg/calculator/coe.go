package calculator

import (
	"dca-platform/pkg/finance"
	"fmt"
)

// CalculateCOE calcula o retorno de um COE (Capital Protegido e/ou Capado)
// initialAmount: Valor aportado
// quotes: Histórico do ativo objeto
// protected: Se true, o valor final nunca é menor que o inicial
// participation: % de participação na alta (ex: 1.0 para 100%)
// capLimit: % máxima de retorno bruto permitida (ex: 0.20 para 20%). Use 0 para sem limite.
func CalculateCOE(quotes []finance.Quote, initialAmount float64, protected bool, participation float64, capLimit float64) StrategyResult {
	if len(quotes) == 0 {
		return StrategyResult{StrategyName: "COE (Sem dados)"}
	}

	startPrice := quotes[0].Close
	endPrice := quotes[len(quotes)-1].Close

	// Rentabilidade do ativo objeto
	assetReturn := (endPrice - startPrice) / startPrice

	// Aplica participação
	grossReturn := assetReturn * participation

	// Aplica Cap (Teto) na ALTA
	if capLimit > 0 && grossReturn > capLimit {
		grossReturn = capLimit
	}

	// Aplica Capital Protegido na BAIXA
	if protected && grossReturn < 0 {
		grossReturn = 0
	}

	finalValue := initialAmount * (1 + grossReturn)
	netReturn := grossReturn * 100 // Em %

	// Nome descritivo
	protStr := "Sem Proteção"
	if protected {
		protStr = "Protegido"
	}
	
	capStr := "Sem Teto"
	if capLimit > 0 {
		capStr = fmt.Sprintf("Cap %.0f%%", capLimit*100)
	}

	strategyName := fmt.Sprintf("COE (%s, Part. %.0f%%, %s)", protStr, participation*100, capStr)

	return StrategyResult{
		StrategyName:     strategyName,
		TotalInvested:    initialAmount,
		FinalValue:       finalValue,
		ReturnPercent:    netReturn,
		TotalAccumulated: 0, // COE não acumula cotas, é um derivativo
	}
}
