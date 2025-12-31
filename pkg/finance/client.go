package finance

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"
)

// Quote representa um preço histórico em uma data
type Quote struct {
	Date  time.Time
	Close float64
}

// Estruturas para parse do JSON da Chart API
type ChartResponse struct {
	Chart struct {
		Result []struct {
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Close []float64 `json:"close"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"chart"`
}

// Client para buscar dados
type Client struct{}

// NewClient cria um novo cliente
func NewClient() *Client {
	return &Client{}
}

// GetHistoricalData busca dados históricos do Yahoo Finance via Chart API JSON
func (c *Client) GetHistoricalData(symbol string, startDate, endDate time.Time, useNative bool) ([]Quote, error) {
	// Lógica para Ativos Sintéticos de Renda Fixa Brasileira
	// Ex: FIXED-BRL-6 -> Renda Fixa 6% a.a. em BRL convertida para USD
	if len(symbol) > 10 && symbol[:10] == "FIXED-BRL-" {
		rateStr := symbol[10:]
		var annualRate float64
		fmt.Sscanf(rateStr, "%f", &annualRate)
		
		return c.getSyntheticFixedIncomeData(annualRate, startDate, endDate, useNative)
	}

	// Lógica para Ações Brasileiras (.SA) - Conversão automática para USD
	if len(symbol) > 3 && symbol[len(symbol)-3:] == ".SA" {
		return c.getBrazilianStockInUSD(symbol, startDate, endDate, useNative)
	}

	period1 := startDate.Unix()
	period2 := endDate.Unix()

	// URL da Chart API (API v8) - geralmente mais permissiva que v7/download
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?period1=%d&period2=%d&interval=1d", symbol, period1, period2)

	return c.fetchRawQuotes(url)
}



// getBrazilianStockInUSD busca a ação em BRL e converte para USD usando o câmbio do dia
func (c *Client) getBrazilianStockInUSD(symbol string, startDate, endDate time.Time, useNative bool) ([]Quote, error) {
	period1 := startDate.Unix()
	period2 := endDate.Unix()
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?period1=%d&period2=%d&interval=1d", symbol, period1, period2)
	
	stockQuotes, err := c.fetchRawQuotes(url)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar %s original: %v", symbol, err)
	}

	// Se o usuário quer moeda original, retornamos direto o preço em Reais
	if useNative {
		return stockQuotes, nil
	}
	
	// 2. Buscar Câmbio (BRL=X)
	exchangeQuotes, err := c.GetHistoricalData("BRL=X", startDate, endDate, false) // recursão evita loop pq simbolo != .SA e FIXED
	if err != nil {
		// Fallback ou erro?
		// Vamos retornar erro, pois o usuário quer comparacao em USD
		return nil, fmt.Errorf("erro ao obter câmbio: %v", err)
	}

	// 3. Cruzar dados e converter e alinhar datas
	// Mapa de câmbio para acesso rápido por data (YYYY-MM-DD)
	exchangeMap := make(map[string]float64)
	for _, q := range exchangeQuotes {
		key := q.Date.Format("2006-01-02")
		exchangeMap[key] = q.Close
	}

	var convertedQuotes []Quote
	for _, sq := range stockQuotes {
		key := sq.Date.Format("2006-01-02")
		rate, ok := exchangeMap[key]
		
		if !ok || rate == 0 {
			continue
		}
		
		convertedPrice := sq.Close / rate
		convertedQuotes = append(convertedQuotes, Quote{
			Date: sq.Date,
			Close: convertedPrice,
		})
	}
	
	return convertedQuotes, nil
}

// fetchRawQuotes encapsula a chamada HTTP básica ao Yahoo para reutilização
func (c *Client) fetchRawQuotes(url string) ([]Quote, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var chartResp ChartResponse
	if err := json.NewDecoder(resp.Body).Decode(&chartResp); err != nil {
		return nil, err
	}

	if len(chartResp.Chart.Result) == 0 {
		return nil, fmt.Errorf("sem dados")
	}

	result := chartResp.Chart.Result[0]
	timestamps := result.Timestamp
	closes := result.Indicators.Quote[0].Close
	
	minLen := len(timestamps)
	if len(closes) < minLen {
		minLen = len(closes)
	}
	
	var quotes []Quote
	for i := 0; i < minLen; i++ {
		if closes[i] == 0 { continue }
		quotes = append(quotes, Quote{
			Date: time.Unix(timestamps[i], 0),
			Close: closes[i],
		})
	}
	return quotes, nil
}

// getSyntheticFixedIncomeData gera dados para um ativo de renda fixa em BRL convertido para USD
func (c *Client) getSyntheticFixedIncomeData(annualRatePercent float64, startDate, endDate time.Time, useNative bool) ([]Quote, error) {
	// 1. Obter histórico do Câmbio (USD/BRL) -> BRL=X
	// Precisamos das datas para saber quais dias de "mercado" existem, mesmo se useNative=true.
	// Usar BRL=X como proxy de dias úteis/mercado é razoável.
	exchangeQuotes, err := c.GetHistoricalData("BRL=X", startDate, endDate, false)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter câmbio para cálculo sintético: %v", err)
	}

	if len(exchangeQuotes) == 0 {
		return nil, fmt.Errorf("sem dados de câmbio para o período")
	}

	// 2. Calcular taxa diária
	// Taxa Anual = (1 + Taxa Diária ^ 252) ou 365. Vamos usar juros compostos simples base 365 para facilitar (crypto roda 24/7).
	// dailyRate = (1 + annualRate)^(1/365) - 1
	dailyRate := pow(1+annualRatePercent/100.0, 1.0/365.0) - 1.0

	var quotes []Quote
	
	// Valor inicial arbitrário em BRL (ex: 100).
	firstDate := exchangeQuotes[0].Date
	
	for _, eq := range exchangeQuotes {
		// Dias passados desde o início da série
		daysPassed := eq.Date.Sub(firstDate).Hours() / 24.0
		if daysPassed < 0 {
			daysPassed = 0
		}
		
		// Rendimento acumulado exato até esta data
		// Value = Initial * (1+Daily)^Days
		accumulatedBRL := 100.0 * pow(1+dailyRate, daysPassed)
		
		var finalValue float64
		if useNative {
			finalValue = accumulatedBRL
		} else {
			// Converter para USD
			rateBRLUSD := eq.Close
			if rateBRLUSD == 0 {
				continue
			}
			finalValue = accumulatedBRL / rateBRLUSD
		}
		
		quotes = append(quotes, Quote{
			Date:  eq.Date,
			Close: finalValue,
		})
	}

	return quotes, nil
}

func pow(x, y float64) float64 {
	return math.Pow(x, y)
}
