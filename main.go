package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

type Cadence int

const (
	Monthly Cadence = iota
	Yearly
)

func main() {
	results := GetTaxDataRange(0, 100000, Yearly)
	WriteTaxResults(results)
}

type TaxResult struct {
	Brutto                   float64 `json:"brutto"`
	GeldwerterVorteil        float64 `json:"geldwertervorteil"`
	Solidaritätszuschlag     float64 `json:"solidaritätszuschlag"`
	Kirchensteuer            float64 `json:"kirchensteuer"`
	Lohnsteuer               float64 `json:"lohnsteuer"`
	Steuern                  float64 `json:"steuern"`
	Rentenversicherung       float64 `json:"rentenversicherung"`
	Arbeitslosenversicherung float64 `json:"arbeitslosenversicherung"`
	Krankenversicherung      float64 `json:"krankenversicherung"`
	Pflegeversicherung       float64 `json:"pflegeversicherung"`
	SozialAbgaben            float64 `json:"sozialabgaben"`
	Netto                    float64 `json:"netto"`
}

// GetTaxDataRange gets tax data from the gross range start to stop in 1000€ increments.
func GetTaxDataRange(start int, stop int, cadence Cadence) []TaxResult {
	var results []TaxResult
	for gross := start; gross <= stop; gross += 1000 {
		results = append(results, GetTaxData(gross, cadence))
	}
	return results
}

// TODO Input is hardcoded
// GetTaxData gets cadence-based tax data for gross income.
func GetTaxData(gross int, cadence Cadence) TaxResult {
	log.Default().Printf("Getting tax data for %d", gross)
	res, err := http.PostForm("https://www.brutto-netto-rechner.info/", url.Values{
		"f_bruttolohn":               {fmt.Sprintf("%d", gross)},
		"f_abrechnungszeitraum":      {"jahr"},
		"f_geld_werter_vorteil":      {"0"},
		"f_abrechnungsjahr":          {"2024"},
		"f_steuerfreibetrag":         {"0"},
		"f_steuerklasse":             {"1"},
		"f_kirche":                   {"nein"},
		"f_bundesland":               {"bayern"},
		"f_alter":                    {"27"},
		"f_kinder":                   {"nein"},
		"f_kinderfreibetrag":         {"0"},
		"f_krankenversicherung":      {"pflichtversichert"},
		"f_private_k":                {""},
		"f_arbeitgeberzuschuss_pkv":  {"ja"},
		"f_KVZ":                      {"1.2"},
		"f_rentenversicherung":       {"pflichtversichert"},
		"f_arbeitslosenversicherung": {"pflichtversichert"},
		"ok":                         {"1"},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	doc, err := html.Parse(res.Body)
	if err != nil {
		log.Fatalln("Error parsing HTML:", err)
	}
	return GetTableData(doc, cadence)
}

// GetTableData parses doc and returns a cadence-based TaxResult.
func GetTableData(doc *html.Node, cadence Cadence) TaxResult {
	regular_row := "right_column"
	final_row := "right_column orange big"

	if cadence == Monthly {
		regular_row = fmt.Sprintf("%v %v", regular_row, "grey_bg")
		final_row = fmt.Sprintf("%v %v", final_row, "grey_bg")
	}
	var results []float64

	// Traverse the HTML tree to find and print the text content of <td> elements
	var extractText func(*html.Node)
	extractText = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "td" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && attr.Val == regular_row || attr.Val == final_row {
					var content string

					if attr.Val == regular_row {
						content = n.FirstChild.Data
					} else if attr.Val == final_row {
						// The 'Netto' amount in the final row is enclosed in another b tag.
						for c := n.FirstChild; c != nil; c = c.NextSibling {
							if c.Data == "b" && c.FirstChild != nil {
								content = c.FirstChild.Data
							}
						}
					}

					// TODO simplify this
					content2 := strings.ReplaceAll(content, "\u00a0", "")
					content2 = strings.ReplaceAll(content2, " ", "")
					content2 = strings.ReplaceAll(content2, "€", "")
					content2 = strings.ReplaceAll(content2, ".", "")
					content2 = strings.ReplaceAll(content2, ",", ".")
					value, err := strconv.ParseFloat(content2, 64)
					if err != nil {
						log.Fatal(err)
					}
					results = append(results, value)
				}

			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractText(c)
		}
	}

	extractText(doc)

	trType := reflect.TypeOf(TaxResult{})
	if len(results) != trType.NumField() {
		log.Fatal("Row amount mismatch!")
	}

	return TaxResult{
		Brutto:                   results[0],
		GeldwerterVorteil:        results[1],
		Solidaritätszuschlag:     results[2],
		Kirchensteuer:            results[3],
		Lohnsteuer:               results[4],
		Steuern:                  results[5],
		Rentenversicherung:       results[6],
		Arbeitslosenversicherung: results[7],
		Krankenversicherung:      results[8],
		Pflegeversicherung:       results[9],
		SozialAbgaben:            results[10],
		Netto:                    results[11],
	}
}

// WriteTaxResults writes results to steuer.jsonl.
func WriteTaxResults(results []TaxResult) {

	file, err := os.Create("steuer.jsonl")
	if err != nil {
		log.Fatalf("Error creating file: %s", err)
	}
	defer file.Close()
	for _, m := range results {
		line, err := json.Marshal(m)
		if err != nil {
			log.Printf("Error encoding JSON: %s", err)
			return
		}
		line = append(line, '\n')

		if _, err := file.Write(line); err != nil {
			log.Printf("Error writing to file: %s", err)
			return
		}
	}

}
