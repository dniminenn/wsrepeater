function prepareData(xData, yDataFunc, traceName, observations) {
    return {
        x: xData,
        y: observations.map(yDataFunc),
        type: "scatter",
        mode: "lines",
        name: traceName,
        line: {
            color: traceName === "Gusts" ? "#ffb86c" : "#8be9fd",
            shape: "spline",
            smoothing: 1.1,
        },
    };
}

function plotData(allObservations) {
    allObservations.sort(
        (a, b) => new Date(a.obsTimeLocal) - new Date(b.obsTimeLocal),
    );

    const times = allObservations.map((obs) => new Date(obs.obsTimeLocal));

    // Plot configurations
    const plots = [
        {
            div: "temperaturePlot",
            yDataFunc: (obs) => obs.metric.tempAvg.toFixed(1), // Celsius
            title: "Temperature",
            yTitle: "Temperature (°C)",
        },
        {
            div: "windPlot",
            yDataFunc: [
                (obs) => obs.metric.windspeedAvg.toFixed(1), // km/h
                (obs) => obs.metric.windgustHigh.toFixed(1), // km/h
            ],
            title: "Wind Speed & Gusts",
            yTitle: "Speed (km/h)",
            traceNames: ["Speed", "Gusts"],
        },
        {
            div: "humidityPlot",
            yDataFunc: (obs) => obs.humidityAvg,
            title: "Humidity",
            yTitle: "Humidity (%)",
        },
        {
            div: "pressurePlot",
            yDataFunc: (obs) =>
                ((obs.metric.pressureMax + obs.metric.pressureMin) / 2).toFixed(
                    1,
                ), // hPa
            title: "Pressure",
            yTitle: "Pressure (hPa)",
        },
        {
            div: "rainPlot",
            yDataFunc: (obs) => obs.metric.precipRate.toFixed(2), // mm
            title: "Rain",
            yTitle: "Precipitation (mm/h)",
        },
        {
            div: "uvPlot",
            yDataFunc: (obs) => obs.uvHigh,
            title: "UV Index",
            yTitle: "UV Index",
        },
    ];

    const plotContainer = document.getElementById("plot-container");
    plotContainer.innerHTML = plots
        .map((plot) => `<div id="${plot.div}" class="plot"></div>`)
        .join("");

    // Plot each graph
    plots.forEach((plotInfo) => {
        const plotElement = document.getElementById(plotInfo.div);
        const plotWidth =
            plotElement.clientWidth || plotElement.parentElement.clientWidth;
        let traces;
        if (Array.isArray(plotInfo.yDataFunc)) {
            traces = plotInfo.yDataFunc.map((yDataFunc, idx) =>
                prepareData(
                    times,
                    yDataFunc,
                    plotInfo.traceNames[idx],
                    allObservations,
                ),
            );
        } else {
            traces = [
                prepareData(
                    times,
                    plotInfo.yDataFunc,
                    plotInfo.yTitle,
                    allObservations,
                ),
            ];
        }

        // Apply dark theme to the layout
        let layout = {
            title: {
                text: plotInfo.title,
                font: {
                    color: "#c7c9cb",
                    family: "Work Sans, sans-serif",
                },
            },
            width: plotWidth * 0.9,
            xaxis: {
                title: {
                    text: "Time",
                    font: {
                        color: "#c7c9cb",
                        family: "Work Sans, sans-serif",
                    },
                },
                tickformat: "%Y-%m-%d %H:%M",
                type: "date",
                gridcolor: "#444",
                zerolinecolor: "#444",
                tickcolor: "#c7c9cb",
                tickfont: {
                    color: "#c7c9cb",
                    family: "Work Sans, sans-serif",
                },
            },
            yaxis: {
                title: {
                    text: plotInfo.yTitle,
                    font: {
                        color: "#c7c9cb",
                        family: "Work Sans, sans-serif",
                    },
                },
                gridcolor: "#444",
                zerolinecolor: "#444",
                tickcolor: "#c7c9cb",
                tickfont: {
                    color: "#c7c9cb",
                    family: "Work Sans, sans-serif",
                },
            },
            plot_bgcolor: "#232530",
            paper_bgcolor: "#232530",
            font: {
                color: "#c7c9cb",
                family: "Work Sans, sans-serif",
            },
            autosize: true,
        };

        Plotly.newPlot(plotInfo.div, traces, layout);
    });
}

function fetchAndPlotAll(days = 7) {
    const url = `${window.location.protocol}//${window.location.hostname}:${window.location.port}/weekly`;

    fetch(url)
        .then((response) => response.json())
        .then((data) => {
            const weeklyData = data.weeklyData;

            // Flatten the weeklyData array to combine all observations into one array
            let allObservations = [];
            weeklyData.slice(0, days).forEach((dayData) => {
                allObservations = allObservations.concat(dayData);
            });

            plotData(allObservations);
        })
        .catch((error) => {
            console.error("Failed to fetch weekly data:", error);
            document.querySelectorAll(".plot").forEach((div) => {
                div.innerHTML = "<p>Failed to load weather data.</p>";
            });
        });
}

document.addEventListener("DOMContentLoaded", function () {
    const viewButtons = document.querySelectorAll(".view-btn");
    viewButtons.forEach((button) => {
        button.addEventListener("click", function () {
            const days = parseInt(button.getAttribute("data-days"));
            fetchAndPlotAll(days);
        });
    });

    // Initial fetch for the default 7-day view
    fetchAndPlotAll(7);

    window.addEventListener("resize", () => {
        fetchAndPlotAll(); // Replot on window resize
    });
});
