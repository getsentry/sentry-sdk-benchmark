package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
)

// Based on script from: https://github.com/tsenart/vegeta/blob/master/lib/plot/assets/plot.html.tpl#L24
const chartScript = `document.getElementById("{{ .ID }}");
  const o = {{ .Opts }}
  const d = {{ .Data }}
  const plot = new Dygraph(container, data, opts);`

// GenerateChart creates a JS snippet that creates a Dygraph chart
// based on given data and options. Dygraph charts require a container
// to attach to, which can be identified by dom element id.
//
// TODO(abhi): Figure out reasonable defaults for dygraph options
func GenerateChart(id string, data []byte, opts DygraphsOpts) (template.JS, error) {
	t, err := template.New(fmt.Sprintf("%s dygraph chart", id)).Parse(chartScript)
	if err != nil {
		return "", err
	}

	optsJSON, err := json.MarshalIndent(&opts, "    ", " ")
	if err != nil {
		return "", err
	}

	var b bytes.Buffer
	t.Execute(&b, ChartData{
		ID:   id,
		Data: template.JS(data),
		Opts: template.JS(optsJSON),
	})

	return template.JS(b.String()), nil
}

var colors = []string{"#2B1D38", "#776589", "#9386A0", "#C6BECF", "#E7E1EC"}

func getLabelColors(labels []string) []string {
	l := len(labels)

	if l > len(colors) {
		panic("not enough colours")
	}

	return colors[:len(labels)]
}

type DygraphsOpts struct {
	Title       string   `json:"title"`
	Labels      []string `json:"labels,omitempty"`
	YLabel      string   `json:"ylabel"`
	XLabel      string   `json:"xlabel"`
	Colors      []string `json:"colors,omitempty"`
	Legend      string   `json:"legend"`
	ShowRoller  bool     `json:"showRoller"`
	LogScale    bool     `json:"logScale"`
	StrokeWidth float64  `json:"strokeWidth"`
}

type ChartData struct {
	ID   string
	Data template.JS
	Opts template.JS
}
