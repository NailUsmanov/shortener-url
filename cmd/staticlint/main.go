package main

import (
	"encoding/json"
	_ "net/http/pprof"
	"os"

	"github.com/NailUsmanov/practicum-shortener-url/cmd/staticlint/osexitanalyzer"
	"github.com/gordonklaus/ineffassign/pkg/ineffassign"
	"github.com/timakin/bodyclose/passes/bodyclose"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
)

// Config — имя файла конфигурации.
const Config = "cmd/staticlint/config.json"

// ConfigData описывает структуру файла конфигурации.
type ConfigData struct {
	Staticcheck []string `json:"staticcheck"`
}

func main() {
	data, err := os.ReadFile(Config)
	if err != nil {
		panic(err)
	}
	var cfgData ConfigData
	if err = json.Unmarshal(data, &cfgData); err != nil {
		panic(err)
	}
	mychecks := []*analysis.Analyzer{
		printf.Analyzer,
		shadow.Analyzer,
		structtag.Analyzer,
	}
	checks := make(map[string]bool)
	for _, v := range cfgData.Staticcheck {
		checks[v] = true
	}
	// Добавляем ВСЕ анализаторы класса SA из staticcheck
	for _, v := range staticcheck.Analyzers {
		if checks[v.Analyzer.Name] {
			mychecks = append(mychecks, v.Analyzer)

		}
	}
	// Добавляем минимум один анализатор из других классов staticcheck (S1000 из класса Simple)
	for _, v := range simple.Analyzers {
		if v.Analyzer.Name == "S1000" {
			mychecks = append(mychecks, v.Analyzer)
			break
		}
	}
	// Добавляем два публичных анализатора
	mychecks = append(mychecks, ineffassign.Analyzer) // Проверка неэффективных присваиваний
	mychecks = append(mychecks, bodyclose.Analyzer)   // Проверяет обработку ошибок
	mychecks = append(mychecks, osexitanalyzer.Analyzer)

	//Создаем мультичекер
	multichecker.Main(
		mychecks...,
	)
}
