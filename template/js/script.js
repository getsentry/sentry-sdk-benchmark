// Based on example from: https://github.com/HdrHistogram/HdrHistogram/blob/9af106907eb6618c4c3fb5ac0da773ad0fee4f13/GoogleChartsExample/plotFiles.html

// Declare constants
const DEFAULT_TICK = { v: 1000000, f: "99.9999%" };
const TICKS = [
  { v: 1, label: "0%" },
  { v: 10, label: "90%" },
  { v: 100, label: "99%" },
  { v: 1000, label: "99.9%" },
  { v: 10000, label: "99.99%" },
  { v: 100000, label: "99.999%" },
  { v: 1000000, label: "99.9999%" },
  { v: 10000000, label: "99.99999%" },
];

const CLASSNAMES = Object.freeze({
  JSON_FORMAT: "jsonFormat",
  HDR: "hdr"
});

// Declare mutable globals
let maxPercentile = DEFAULT_TICK.v;
let chartData = null;
let chart = null;

document.addEventListener("DOMContentLoaded", drawInitialChart);

function drawInitialChart() {
  const histos = [];
  const names = [];
  const hdrNodes = document.querySelectorAll(getClassName(CLASSNAMES.HDR));
  hdrNodes.forEach((node) => {
    // name
    names.push(node.getAttribute("data-name"));
    // actual data
    histos.push(node.innerHTML);
  });

  drawChart(names, histos);
  createEnvDetails();

  formatJSON();
}

function setChartData(names, histos) {
  while (names.length < histos.length) {
    names.push("Unknown");
  }

  let series = [];
  for (let i = 0; i < histos.length; i++) {
    series = appendDataSeries(histos[i], names[i], series);
  }

  return series
}

function drawChart(names, histos) {
  const [_, ...data] = setChartData(names, histos);
  const options = {
    title: "Latency by Percentile Distribution",
    height: 480,
    hAxis: {
      title: "Percentile",
      minValue: 1,
      logScale: true,
      ticks: TICKS,
      viewWindowMode: "explicit",
      viewWindow: {
        max: maxPercentile,
        min: 1,
      },
    },
    vAxis: { title: "Latency (ms)", minValue: 0 },
    legend: { position: "bottom" },
  };

  new Dygraph(document.getElementById('percentileLatency'), data, {
    title: 'Latency by Percentile Distribution',
    logscale: true,
    ylabel: 'Latency (ms)',
    xlabel: 'Percentile',
    legend: 'always',
    labels: ["Percentile"].concat(names),
    axes: {
      x: {
        ticker: () => TICKS,
        logscale: true,
      },
    },
    showRoller: true,
    strokeWidth: 1.3,
    drawAxesAtZero: true,
  });
}

function appendDataSeries(histo, name, dataSeries) {
  let series;
  let seriesCount;
  if (dataSeries.length == 0) {
    series = [["X", name]];
    seriesCount = 1;
  } else {
    series = dataSeries;
    series[0].push(name);
    seriesCount = series[0].length - 1;
  }

  const lines = histo.split("\n");

  let seriesIndex = 1;
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i].trim();
    const values = line.trim().split(/[ ]+/);

    if (line[0] != "#" && values.length == 4) {
      const y = parseFloat(values[0]);
      const x = parseFloat(values[3]);

      if (!isNaN(x) && !isNaN(y)) {
        if (seriesIndex >= series.length) {
          series.push([x]);
        }

        while (series[seriesIndex].length < seriesCount) {
          series[seriesIndex].push(null);
        }

        series[seriesIndex].push(y);
        seriesIndex++;
      }
    }
  }

  while (seriesIndex < series.length) {
    series[seriesIndex].push(null);
    seriesIndex++;
  }

  return series;
}

function createEnvDetails() {
  const node = document.getElementById("requestEnv");
  const items = node.innerHTML.split("\n").filter((i) => i !== "");

  const formatNode = document.getElementById("formatEnv");
  formatNode.innerHTML = "";

  items.forEach((item) => {
    const preEle = document.createElement("pre");
    preEle.className = CLASSNAMES.JSON_FORMAT;
    preEle.innerHTML = item;
    formatNode.appendChild(preEle);
  });
}

function formatJSON() {
  const nodes = document.querySelectorAll(getClassName(CLASSNAMES.JSON_FORMAT));
  nodes.forEach((node) => {
    node.innerHTML = JSON.stringify(JSON.parse(node.innerHTML), null, 4);
  });
}

// Reacts to Slider Input: Updates Chart
function showValue(newValue) {
  const newPercentile = Math.pow(10, newValue);
  const percentile = 100.0 - 100.0 / newPercentile;
  document.getElementById("percentileRange").innerHTML = percentile + "%";
  maxPercentile = newPercentile;
  drawChart();
  return { typed: "" };
}

function copy(className, runName) {
  const hdrNodes = document.querySelectorAll(`.${className}`);
  hdrNodes.forEach((node) => {
    if (node.getAttribute("data-name") === runName) {
      navigator.clipboard.writeText(node.innerHTML);
      const snack = document.getElementById("snackbar");
      snackbar.innerHTML = `Copied ${className} for ${runName}`;
      // Add the "show" class to DIV
      snackbar.className = "show";
      setTimeout(() => {
        snackbar.className = snackbar.className.replace("show", "");
        snackbar.innerHTML = "";
      }, 1000);
    }
  });
}

const getClassName = (name) => `.${name}`;
