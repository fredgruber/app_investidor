package main

import (
	"dca-platform/pkg/calculator"
	"dca-platform/pkg/finance"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type PageData struct {
	StartDate   string
	EndDate     string
	Amount      string
	Frequency   string
	Results     []calculator.StrategyResult
	BestStrategy string
	Error       string
}

func main() {
	// Debug: Imprimir diretório atual
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Diretório atual de execução:", dir)

	// Debug: Verificar se arquivo existe
	if _, err := os.Stat("templates/index.html"); os.IsNotExist(err) {
		fmt.Println("CRÍTICO: Arquivo templates/index.html NÃO encontrado no diretório atual.")
	} else {
		fmt.Println("OK: Arquivo templates/index.html encontrado.")
	}

	// Servir arquivos estáticos (CSS)
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", handleHome)
	http.HandleFunc("/simulate", handleSimulate)

	fmt.Println("Servidor rodando em http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	// Valores padrão
	data := PageData{
		StartDate: "2017-01-01",
		EndDate:   time.Now().Format("2006-01-02"),
		Amount:    "100",
		Frequency: "monthly",
	}
	renderTemplate(w, data)
}

func handleSimulate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	startDateStr := r.FormValue("startDate")
	endDateStr := r.FormValue("endDate")
	amountStr := r.FormValue("amount")
	freqStr := r.FormValue("frequency")

	data := PageData{
		StartDate: startDateStr,
		EndDate:   endDateStr,
		Amount:    amountStr,
		Frequency: freqStr,
	}

	// Conversões
	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		data.Error = "Data de início inválida."
		renderTemplate(w, data)
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		data.Error = "Data de fim inválida."
		renderTemplate(w, data)
		return
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		data.Error = "Valor de investimento inválido."
		renderTemplate(w, data)
		return
	}

	// Buscar dados
	client := finance.NewClient()
	
	btcData, err := client.GetHistoricalData("BTC-USD", startDate, endDate)
	if err != nil {
		data.Error = "Erro ao buscar dados do Bitcoin: " + err.Error()
		renderTemplate(w, data)
		return
	}

	goldData, err := client.GetHistoricalData("GC=F", startDate, endDate)
	if err != nil {
		fmt.Println("Aviso: Ouro data error:", err)
	}

	sp500Data, err := client.GetHistoricalData("^GSPC", startDate, endDate)
	if err != nil {
		fmt.Println("Aviso: SP500 data error:", err)
	}

	// Calcular DCA BTC
	var freq calculator.Frequency
	switch freqStr {
	case "daily":
		freq = calculator.Daily
	case "weekly":
		freq = calculator.Weekly
	default:
		freq = calculator.Monthly
	}

	dcaResult := calculator.CalculateDCA(btcData, amount, freq)
	
	totalInvested := dcaResult.TotalInvested
	
	lumpSumBTC := calculator.CalculateLumpSum(btcData, totalInvested, "Lump Sum Bitcoin")
	lumpSumGold := calculator.CalculateLumpSum(goldData, totalInvested, "Lump Sum Ouro")
	lumpSumSP500 := calculator.CalculateLumpSum(sp500Data, totalInvested, "Lump Sum S&P 500")

	// Compilar resultados
	results := []calculator.StrategyResult{dcaResult, lumpSumBTC, lumpSumGold, lumpSumSP500}
	data.Results = results

	// Encontrar melhor estratégia
	bestReturn := -999999.0
	bestName := ""
	for _, res := range results {
		if res.ReturnPercent > bestReturn {
			bestReturn = res.ReturnPercent
			bestName = res.StrategyName
		}
	}
	data.BestStrategy = bestName

	renderTemplate(w, data)
}

func renderTemplate(w http.ResponseWriter, data PageData) {
	// Como main.go está na raiz, templates/index.html funcionará
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Erro no template (verifique se está rodando da raiz do projeto): "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}
