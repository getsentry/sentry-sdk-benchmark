package main

import (
	"bytes"
	"fmt"
	"html/template"
)

// Based on script from: https://github.com/tsenart/vegeta/blob/master/lib/plot/assets/plot.html.tpl#L24
const chartHTML = `<script>
  document.addEventListener("DOMContentLoaded", () => {
	const c = document.getElementById("{{ .ID }}");
	const d = {{ .Data }};
	const o = {{ .Opts }};
	new Dygraph(c, d, o);
  });
</script>
<div id="{{ .ID }}" class="dygraphChart"></div>`

// GenerateChart creates a JS snippet that creates a Dygraph chart
// based on given data and options. Dygraph charts require a container
// to attach to, which can be identified by dom element id.
//
// TODO(abhi): Figure out reasonable defaults for dygraph options
func GenerateChart(id string, data []byte, opts DygraphsOpts) (template.HTML, error) {
	t, err := template.New(fmt.Sprintf("%s dygraph chart", id)).Parse(chartHTML)
	if err != nil {
		return "", err
	}

	var b bytes.Buffer
	err = t.Execute(&b, ChartData{
		ID:   id,
		Data: template.JS(data),
		Opts: opts,
	})
	if err != nil {
		return "", err
	}

	return template.HTML(b.String()), nil
}

// DygraphsOpts configures options for a Dygraph Chart
// See: https://dygraphs.com/options.html
type DygraphsOpts struct {
	Title       string   `json:"title"`
	Labels      []string `json:"labels,omitempty"`
	YLabel      string   `json:"ylabel"`
	XLabel      string   `json:"xlabel"`
	Colors      []string `json:"colors,omitempty"`
	Legend      string   `json:"legend"`
	ShowRoller  bool     `json:"showRoller"`
	StrokeWidth float64  `json:"strokeWidth"`
	Width       int      `json:"width,omitempty"`
	RollPeriod  int      `json:"rollPeriod,omitempty"`
}

type ChartData struct {
	ID   string
	Data template.JS
	Opts DygraphsOpts
}
