document.addEventListener("DOMContentLoaded", function () {
    const dailyDataUrl = `${window.location.protocol}//${window.location.hostname}:${window.location.port}/wutoday`;
    const forecastUrl = `${window.location.protocol}//${window.location.hostname}:${window.location.port}/rss/city/nb-17_e.xml`;

    function updateDailyData() {
        fetch(dailyDataUrl)
            .then((response) => {
                if (!response.ok) {
                    throw new Error("Network response was not ok");
                }
                return response.json();
            })
            .then((data) => {
                const dailyHistory = data.dailyHistory.observations[0];
                const imperial = dailyHistory.imperial;

                // Convert imperial to metric units
                const tempHighC = (((imperial.tempHigh - 32) * 5) / 9).toFixed(
                    1,
                );
                const tempLowC = (((imperial.tempLow - 32) * 5) / 9).toFixed(1);
                const windspeedHighKmh = (
                    imperial.windspeedHigh * 1.60934
                ).toFixed(1);
                const windgustHighKmh = (
                    imperial.windgustHigh * 1.60934
                ).toFixed(1);
                const pressureMaxHpa = (imperial.pressureMax * 33.8639).toFixed(
                    2,
                );
                const pressureMinHpa = (imperial.pressureMin * 33.8639).toFixed(
                    2,
                );

                // Get the current values from the DOM
                const currentTempC = parseFloat(
                    document.getElementById("temp").textContent,
                );
                const currentWindSpeedKmh = parseFloat(
                    document.getElementById("windSpeed").textContent,
                );
                const currentWindGustKmh = parseFloat(
                    document.getElementById("windGust").textContent,
                );
                const currentPressureHpa = parseFloat(
                    document.getElementById("baro").textContent,
                );
                const currentUv = parseFloat(
                    document.getElementById("uv").textContent,
                );

                // Update daily temperature values
                document.getElementById("tempHigh").textContent = Math.max(
                    tempHighC,
                    currentTempC,
                ).toFixed(1);
                document.getElementById("tempLow").textContent = Math.min(
                    tempLowC,
                    currentTempC,
                ).toFixed(1);

                // Update daily wind values
                document.getElementById("windSpeedMax").textContent = Math.max(
                    windspeedHighKmh,
                    currentWindSpeedKmh,
                ).toFixed(1);
                document.getElementById("windGustMax").textContent = Math.max(
                    windgustHighKmh,
                    currentWindGustKmh,
                ).toFixed(1);

                // Update daily pressure values
                document.getElementById("pressureMax").textContent = Math.max(
                    pressureMaxHpa,
                    currentPressureHpa,
                ).toFixed(2);
                document.getElementById("pressureMin").textContent = Math.min(
                    pressureMinHpa,
                    currentPressureHpa,
                ).toFixed(2);

                // Update daily UV value
                const uvHigh = Math.max(dailyHistory.uvHigh, currentUv);
                document.getElementById("uvHigh").textContent = uvHigh;
            })
            .catch((error) => {
                console.error("Error fetching or processing data:", error);
            });
    }

    function updateForecastData() {
        fetch(forecastUrl)
            .then((response) => response.text())
            .then((str) =>
                new window.DOMParser().parseFromString(str, "application/xml"),
            )
            .then((data) => {
                const entries = data.querySelectorAll("entry");
                let forecastHtml = "<table><tr>";
                let dayPart = "";
                let nightPart = "";
                let cardCounter = 0;
                let currentDate = "";

                entries.forEach((entry, index) => {
                    if (cardCounter >= 4) {
                        // Stop processing if we've reached 4 days
                        return;
                    }

                    const category = entry
                        .querySelector("category")
                        .getAttribute("term");

                    if (category === "Weather Forecasts") {
                        const title = entry.querySelector("title").textContent;

                        // Extract day part (e.g., "Monday" or "Monday night")
                        const match = title.match(/(\w+)\s?(night)?:/);
                        const newDate = match ? match[1] : "";

                        // If new day starts or it's the first entry, reset the parts
                        if (newDate !== currentDate) {
                            if (dayPart || nightPart) {
                                forecastHtml += `<td class="forecast-card">
                                                    <div class="forecast-header">${currentDate}</div>
                                                    <div class="forecast-day">${dayPart}</div>
                                                    <div class="forecast-night">${nightPart}</div>
                                                 </td>`;
                                cardCounter++;
                            }
                            dayPart = "";
                            nightPart = "";
                            currentDate = newDate;
                        }

                        // Determine if it's a day or night entry
                        if (title.toLowerCase().includes("night")) {
                            nightPart = title.replace(
                                `${currentDate} night: `,
                                "<i>Night:</i> ",
                            );
                        } else {
                            dayPart = title.replace(`${currentDate}: `, "");
                        }
                    }
                });

                // Add the final card if there's anything left
                if ((dayPart || nightPart) && cardCounter < 4) {
                    forecastHtml += `<td class="forecast-card">
                                        <div class="forecast-header">${currentDate}</div>
                                        <div class="forecast-day">${dayPart}</div>
                                        <div class="forecast-night">${nightPart}</div>
                                     </td>`;
                }

                forecastHtml += "</tr></table>";

                document.getElementById("forecast").innerHTML = forecastHtml;
            })
            .catch((error) => {
                console.error(
                    "Error fetching or processing forecast data:",
                    error,
                );
            });
    }

    updateDailyData();
    updateForecastData();
    setInterval(() => {
        updateDailyData();
        updateForecastData();
    }, 600000);
});
