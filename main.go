package main

import (
	"dca-platform/pkg/calculator"
	"dca-platform/pkg/finance"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"
)


// Estrutura para definir opções de ativos
type AssetOption struct {
	Symbol   string
	Name     string
	Category string
}

// Lista Global de Ativos Suportados
var SupportedAssets = []AssetOption{
	// Cripto
	{"BTC-USD", "Bitcoin (BTC)", "Cripto"},
	{"ETH-USD", "Ethereum (ETH)", "Cripto"},
	{"SOL-USD", "Solana (SOL)", "Cripto"},
	
	// Commodities / Índices
	{"GC=F", "Ouro (Gold)", "Commodities"},
	{"^GSPC", "S&P 500", "Indices"},
	{"^IXIC", "Nasdaq 100", "Indices"},

	// Brasil (ADRs)
	{"EWZ", "iShares MSCI Brazil ETF", "Brasil"},
	{"PBR", "Petrobras (PBR)", "Brasil"},
	{"VALE", "Vale (VALE)", "Brasil"},
	{"ITUB", "Itaú Unibanco (ITUB)", "Brasil"},
	{"NU", "Nubank (NU)", "Brasil"},

	// Renda Fixa BRL (Sintética USD)
	{"FIXED-BRL-6.17", "Poupança BR (Est. 6.17% a.a.)", "Brasil RF"},
	{"FIXED-BRL-10.0", "Tesouro Selic (Est. 10% a.a.)", "Brasil RF"},
	{"FIXED-BRL-12.0", "CDB Pré (Est. 12% a.a.)", "Brasil RF"},

	// USA Tech / Stocks
	{"AAPL", "Apple (AAPL)", "EUA"},
	{"MSFT", "Microsoft (MSFT)", "EUA"},
	{"GOOGL", "Google (GOOGL)", "EUA"},
	{"AMZN", "Amazon (AMZN)", "EUA"},
	{"TSLA", "Tesla (TSLA)", "EUA"},
	{"NVDA", "NVIDIA (NVDA)", "EUA"},
	{"META", "Meta (Facebook)", "EUA"},
}

type PageData struct {
	StartDate     string
	EndDate       string
	Amount        string
	Frequency     string
	Assets        []AssetOption
	
	// Estado dos checkboxes
	SelectedDCA   map[string]bool
	SelectedLS    map[string]bool
	
	// Estado do Custom Asset
	CustomTicker  string
	CustomDCA     bool
	CustomLS      bool

	Results       []calculator.StrategyResult
	BestStrategy  string
	Error         string
}

func main() {
	// Debug: Imprimir diretório atual
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Diretório atual de execução:", dir)

	// Servir arquivos estáticos (CSS)
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", handleHome)
	http.HandleFunc("/simulate", handleSimulate)

	fmt.Println("Servidor rodando em http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		StartDate: "2017-01-01",
		EndDate:   time.Now().Format("2006-01-02"),
		Amount:    "100",
		Frequency: "monthly",
		Assets:    SupportedAssets,
		SelectedDCA: map[string]bool{
			"BTC-USD": true,
		},
		SelectedLS: map[string]bool{
			"BTC-USD": true,
		},
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
	
	r.ParseForm()
	dcaAssets := r.Form["dca_assets"]
	lsAssets := r.Form["ls_assets"]
	
	customTicker := r.FormValue("custom_ticker")
	customDCA := r.FormValue("custom_dca") == "on"
	customLS := r.FormValue("custom_ls") == "on"
	
	// Adicionar ativo customizado às listas se selecionado
	if customTicker != "" {
		// Normalizar ticker (uppercase)
		// Mas Go não tem strings.ToUpper fácil sem import "strings", vou deixar como está ou adicionar import
		// Assumindo usuário digita certo ou Yahoo resolve.
		if customDCA {
			dcaAssets = append(dcaAssets, customTicker)
		}
		if customLS {
			lsAssets = append(lsAssets, customTicker)
		}
	}

	// Validar inputs
	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		renderError(w, "Data de início inválida.")
		return
	}
	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		renderError(w, "Data de fim inválida.")
		return
	}
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		renderError(w, "Valor inválido.")
		return
	}

	// Reconstruir mapas de seleção
	selDca := make(map[string]bool)
	for _, s := range dcaAssets { selDca[s] = true }
	
	selLs := make(map[string]bool)
	for _, s := range lsAssets { selLs[s] = true }

	data := PageData{
		StartDate:    startDateStr,
		EndDate:      endDateStr,
		Amount:       amountStr,
		Frequency:    freqStr,
		Assets:       SupportedAssets,
		SelectedDCA:  selDca,
		SelectedLS:   selLs,
		CustomTicker: customTicker,
		CustomDCA:    customDCA,
		CustomLS:     customLS,
	}

	if len(dcaAssets) == 0 && len(lsAssets) == 0 {
		data.Error = "Selecione pelo menos um ativo (DCA ou Lump Sum)."
		renderTemplate(w, data)
		return
	}

	client := finance.NewClient()
	var results []calculator.StrategyResult

	// Precisamos saber o TotalInvested padrão para o Lump Sum
	// Para isso, simulamos um DCA em qualquer ativo (ou sem ativo, mas CalculateDCA pede dados)
	// Vamos usar uma logica: se tiver algum DCA selecionado, usamos o TotalInvested dele.
	// Se SÓ tiver Lump Sum, precisamos calcular o TotalInvested teórico baseado nas datas.
	
	// Para simplificar: Calculamos o tempo de investimento.
	// Mas a função CalculateDCA já faz isso perfeitamente considerando dias úteis se usarmos dados reais.
	// Vamos pegar dados de um ativo "base" (BTC) apenas para calcular o calendário de pagamentos, se necessário.
	// Ou melhor: Calcular para cada ativo selecionado independentemente.
	
	var theoreticalTotalInvested float64 = 0
	calculatedTotal := false

	var freq calculator.Frequency
	switch freqStr {
	case "daily":
		freq = calculator.Daily
	case "weekly":
		freq = calculator.Weekly
	default:
		freq = calculator.Monthly
	}

	// Processar DCA Assets
	for _, symbol := range dcaAssets {
		histData, err := client.GetHistoricalData(symbol, startDate, endDate)
		if err != nil {
			fmt.Printf("Erro dados %s: %v\n", symbol, err)
			continue
		}
		
		dcaRes := calculator.CalculateDCA(histData, amount, freq)
		dcaRes.StrategyName = fmt.Sprintf("DCA %s", getAssetName(symbol))
		results = append(results, dcaRes)
		
		if !calculatedTotal {
			theoreticalTotalInvested = dcaRes.TotalInvested
			calculatedTotal = true
		}
	}

	// Se não tivemos nenhum DCA, precisamos calcular o TotalInvested para o Lump Sum.
	// Vamos pegar dados do primeiro ativo LS para ter o calendário.
	if !calculatedTotal && len(lsAssets) > 0 {
		// Pegar dados do primeiro LS para calcular as datas
		histData, err := client.GetHistoricalData(lsAssets[0], startDate, endDate)
		if err == nil {
			// Simular DCA fantasma só para pegar o valor investido
			dummy := calculator.CalculateDCA(histData, amount, freq)
			theoreticalTotalInvested = dummy.TotalInvested
			calculatedTotal = true
		}
	}

	// Processar Lump Sum Assets
	for _, symbol := range lsAssets {
		histData, err := client.GetHistoricalData(symbol, startDate, endDate)
		if err != nil {
			fmt.Printf("Erro dados %s: %v\n", symbol, err)
			continue
		}
		
		// Lump Sum assume investir TUDO no início.
		// Qual valor? O mesmo que seria gasto no DCA (theoreticalTotalInvested).
		lsRes := calculator.CalculateLumpSum(histData, theoreticalTotalInvested, fmt.Sprintf("Lump Sum %s", getAssetName(symbol)))
		results = append(results, lsRes)
	}

	// Ordenar
	sort.Slice(results, func(i, j int) bool {
		return results[i].ReturnPercent > results[j].ReturnPercent
	})

	data.Results = results

	if len(results) > 0 {
		data.BestStrategy = results[0].StrategyName
	}

	renderTemplate(w, data)
}

func renderError(w http.ResponseWriter, msg string) {
	data := PageData{
		Error:  msg,
		Assets: SupportedAssets,
	}
	renderTemplate(w, data)
}

func getAssetName(symbol string) string {
	for _, a := range SupportedAssets {
		if a.Symbol == symbol {
			// Retornar nome curto ou o proprio nome
			return a.Name
		}
	}
	return symbol
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
