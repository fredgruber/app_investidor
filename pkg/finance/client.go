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
func (c *Client) GetHistoricalData(symbol string, startDate, endDate time.Time) ([]Quote, error) {
	// Lógica para Ativos Sintéticos de Renda Fixa Brasileira
	// Ex: FIXED-BRL-6 -> Renda Fixa 6% a.a. em BRL convertida para USD
	if len(symbol) > 10 && symbol[:10] == "FIXED-BRL-" {
		rateStr := symbol[10:]
		var annualRate float64
		fmt.Sscanf(rateStr, "%f", &annualRate)
		
		return c.getSyntheticFixedIncomeData(annualRate, startDate, endDate)
	}

	period1 := startDate.Unix()
	period2 := endDate.Unix()

	// URL da Chart API (API v8) - geralmente mais permissiva que v7/download
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?period1=%d&period2=%d&interval=1d", symbol, period1, period2)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %v", err)
	}
	// User-Agent de navegador moderno
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição HTTP: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erro ao obter dados do Yahoo Finance (Chart API): status %d", resp.StatusCode)
	}

	var chartResp ChartResponse
	if err := json.NewDecoder(resp.Body).Decode(&chartResp); err != nil {
		return nil, fmt.Errorf("erro ao decodificar JSON: %v", err)
	}

	if len(chartResp.Chart.Result) == 0 {
		return nil, fmt.Errorf("nenhum resultado retornado para o símbolo %s", symbol)
	}

	result := chartResp.Chart.Result[0]
	timestamps := result.Timestamp
	closes := result.Indicators.Quote[0].Close

	if len(timestamps) != len(closes) {
		// As vezes pode haver mismatch se houver nulos, mas geralmente é alinhado
		// Vamos prevenir panic
		minLen := len(timestamps)
		if len(closes) < minLen {
			minLen = len(closes)
		}
		timestamps = timestamps[:minLen]
		closes = closes[:minLen]
	}

	var quotes []Quote
	for i, ts := range timestamps {
		// Às vezes o valor é 0 ou null no JSON (que vira 0 no float64 se omitido, mas ponteiro resolveria)
		// Para simplificar, assumimos que 0.0 é inválido para estes ativos
		price := closes[i]
		if price == 0 {
			continue
		}

		quotes = append(quotes, Quote{
			Date:  time.Unix(ts, 0),
			Close: price,
		})
	}

	return quotes, nil
}

// getSyntheticFixedIncomeData gera dados para um ativo de renda fixa em BRL convertido para USD
func (c *Client) getSyntheticFixedIncomeData(annualRatePercent float64, startDate, endDate time.Time) ([]Quote, error) {
	// 1. Obter histórico do Câmbio (USD/BRL) -> BRL=X
	// BRL=X significa "Quantos Reais valem 1 Dólar"
	exchangeQuotes, err := c.GetHistoricalData("BRL=X", startDate, endDate)
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
	
	// Valor inicial arbitrário em BRL (ex: 100). O valor absoluto não importa para o retorno %, 
	// mas para o DCA/LumpSum importa a série de preços.
	// Vamos criar um "índice" que começa em 100.
	
	// Precisamos alinhar as datas. Vamos iterar sobre as datas do câmbio.
	// Assumimos que o rendimento corre todo dia que tem cotação de câmbio.
	
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
		
		// Converter para USD
		// Se 1 USD = X BRL, então Valor USD = Valor BRL / X
		rateBRLUSD := eq.Close
		if rateBRLUSD == 0 {
			continue
		}
		
		valueUSD := accumulatedBRL / rateBRLUSD
		
		quotes = append(quotes, Quote{
			Date:  eq.Date,
			Close: valueUSD,
		})
	}

	return quotes, nil
}

func pow(x, y float64) float64 {
	return math.Pow(x, y)
}
